package helpers

import (
	"fmt"
	"reflect"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type sidecarPodFixtureBuilder struct {
	name, namespace                                               string
	sidecarContainerName, sidecarImageRepository, sidecarImageTag string
	initContainerName                                             string
	podAnnotations                                                map[string]string
	podLabels                                                     map[string]string
	podStatusPhase                                                v1.PodPhase
	conditionStatus                                               v1.ConditionStatus
	deletionTimestamp                                             *metav1.Time
	hostNetwork                                                   bool
}

func NewSidecarPodBuilder() *sidecarPodFixtureBuilder {
	return &sidecarPodFixtureBuilder{
		name:                   "app",
		namespace:              "custom",
		sidecarContainerName:   "istio-proxy",
		sidecarImageRepository: "istio/proxyv2",
		sidecarImageTag:        "1.10.0",
		initContainerName:      "istio-init",
		podAnnotations:         map[string]string{"sidecar.istio.io/status": "{\"containers\":[\"istio-proxy\"]}"},
		podLabels:              map[string]string{},
		podStatusPhase:         "Running",
		conditionStatus:        "True",
		hostNetwork:            false,
	}
}

func (r *sidecarPodFixtureBuilder) SetName(value string) *sidecarPodFixtureBuilder {
	r.name = value
	return r
}

func (r *sidecarPodFixtureBuilder) SetNamespace(value string) *sidecarPodFixtureBuilder {
	r.namespace = value
	return r
}

func (r *sidecarPodFixtureBuilder) SetPodStatusPhase(value v1.PodPhase) *sidecarPodFixtureBuilder {
	r.podStatusPhase = value
	return r
}

func (r *sidecarPodFixtureBuilder) SetPodAnnotations(value map[string]string) *sidecarPodFixtureBuilder {
	r.podAnnotations = value
	return r
}

func (r *sidecarPodFixtureBuilder) SetPodLabels(value map[string]string) *sidecarPodFixtureBuilder {
	r.podLabels = value
	return r
}

func (r *sidecarPodFixtureBuilder) SetPodHostNetwork() *sidecarPodFixtureBuilder {
	r.hostNetwork = true
	return r
}

func (r *sidecarPodFixtureBuilder) SetInitContainer(value string) *sidecarPodFixtureBuilder {
	r.initContainerName = value
	return r
}

func (r *sidecarPodFixtureBuilder) SetSidecarImageRepository(value string) *sidecarPodFixtureBuilder {
	r.sidecarImageRepository = value
	return r
}

func (r *sidecarPodFixtureBuilder) SetSidecarImageTag(value string) *sidecarPodFixtureBuilder {
	r.sidecarImageTag = value
	return r
}

func (r *sidecarPodFixtureBuilder) SetSidecarContainerName(value string) *sidecarPodFixtureBuilder {
	r.sidecarContainerName = value
	return r
}

func (r *sidecarPodFixtureBuilder) DisableSidecar() *sidecarPodFixtureBuilder {
	r.sidecarContainerName = "workload"
	r.sidecarImageRepository = "image"
	r.sidecarImageTag = "1.0"
	r.initContainerName = "customer-init"
	r.podAnnotations = map[string]string{}
	return r
}

func (r *sidecarPodFixtureBuilder) SetConditionStatus(value v1.ConditionStatus) *sidecarPodFixtureBuilder {
	r.conditionStatus = value
	return r
}

func (r *sidecarPodFixtureBuilder) SetDeletionTimestamp(value time.Time) *sidecarPodFixtureBuilder {
	r.deletionTimestamp = &metav1.Time{Time: value}
	return r
}

func (r *sidecarPodFixtureBuilder) Build() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.name,
			Namespace: r.namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet"},
			},
			Annotations:       r.podAnnotations,
			Labels:            r.podLabels,
			DeletionTimestamp: r.deletionTimestamp,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: v1.PodStatus{
			Phase: r.podStatusPhase,
			Conditions: []v1.PodCondition{
				{
					Type:   "Ready",
					Status: r.conditionStatus,
				},
			},
		},
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{
				{
					Name:  r.initContainerName,
					Image: r.initContainerName,
				},
			},
			Containers: []v1.Container{
				{
					Name:  "workload-container",
					Image: "workload-image:1.0",
				},
				{
					Name:  r.sidecarContainerName,
					Image: fmt.Sprintf(`%s:%s`, r.sidecarImageRepository, r.sidecarImageTag)},
			},
			HostNetwork: r.hostNetwork,
		},
	}
}

func FixPodWithoutSidecar(name, namespace string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet"},
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: v1.PodStatus{
			Phase: "Running",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "workload-container",
					Image: "workload-image:1.0",
				},
			},
		},
	}
}

func FixNamespaceWith(name string, labels map[string]string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

func Clone(oldObj interface{}) interface{} {
	newObj := reflect.New(reflect.TypeOf(oldObj).Elem())
	oldVal := reflect.ValueOf(oldObj).Elem()
	newVal := newObj.Elem()
	for i := 0; i < oldVal.NumField(); i++ {
		newValField := newVal.Field(i)
		if newValField.CanSet() {
			newValField.Set(oldVal.Field(i))
		}
	}

	return newObj.Interface()
}
