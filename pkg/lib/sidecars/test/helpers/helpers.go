package helpers

import (
	"fmt"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SidecarPodFixtureBuilder struct {
	name, namespace                                               string
	sidecarContainerName, sidecarImageRepository, sidecarImageTag string
	initContainerName                                             string
	podAnnotations                                                map[string]string
	podLabels                                                     map[string]string
	podStatusPhase                                                v1.PodPhase
	conditionStatus                                               v1.ConditionStatus
	deletionTimestamp                                             *metav1.Time
	hostNetwork                                                   bool
	ownerReference                                                metav1.OwnerReference
	resources                                                     v1.ResourceRequirements
	finalizers                                                    []string
}

var DefaultSidecarResources = v1.ResourceRequirements{
	Limits: v1.ResourceList{
		v1.ResourceCPU:    resource.MustParse("100m"),
		v1.ResourceMemory: resource.MustParse("200Mi"),
	},
	Requests: v1.ResourceList{
		v1.ResourceCPU:    resource.MustParse("200m"),
		v1.ResourceMemory: resource.MustParse("400Mi"),
	},
}

func NewSidecarPodBuilder() *SidecarPodFixtureBuilder {
	resources := DefaultSidecarResources.DeepCopy()
	return &SidecarPodFixtureBuilder{
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
		ownerReference:         metav1.OwnerReference{Kind: "ReplicaSet"},
		resources:              *resources,
	}
}

func (r *SidecarPodFixtureBuilder) SetName(value string) *SidecarPodFixtureBuilder {
	r.name = value
	return r
}

func (r *SidecarPodFixtureBuilder) SetNamespace(value string) *SidecarPodFixtureBuilder {
	r.namespace = value
	return r
}

func (r *SidecarPodFixtureBuilder) SetPodStatusPhase(value v1.PodPhase) *SidecarPodFixtureBuilder {
	r.podStatusPhase = value
	return r
}

func (r *SidecarPodFixtureBuilder) SetPodAnnotations(value map[string]string) *SidecarPodFixtureBuilder {
	r.podAnnotations = value
	return r
}

func (r *SidecarPodFixtureBuilder) SetPodLabels(value map[string]string) *SidecarPodFixtureBuilder {
	r.podLabels = value
	return r
}

func (r *SidecarPodFixtureBuilder) SetPodHostNetwork() *SidecarPodFixtureBuilder {
	r.hostNetwork = true
	return r
}

func (r *SidecarPodFixtureBuilder) SetInitContainer(value string) *SidecarPodFixtureBuilder {
	r.initContainerName = value
	return r
}

func (r *SidecarPodFixtureBuilder) SetSidecarImageRepository(value string) *SidecarPodFixtureBuilder {
	r.sidecarImageRepository = value
	return r
}

func (r *SidecarPodFixtureBuilder) SetSidecarImageTag(value string) *SidecarPodFixtureBuilder {
	r.sidecarImageTag = value
	return r
}

func (r *SidecarPodFixtureBuilder) SetSidecarContainerName(value string) *SidecarPodFixtureBuilder {
	r.sidecarContainerName = value
	return r
}

func (r *SidecarPodFixtureBuilder) SetOwnerReference(or metav1.OwnerReference) *SidecarPodFixtureBuilder {
	r.ownerReference = or
	return r
}

func (r *SidecarPodFixtureBuilder) DisableSidecar() *SidecarPodFixtureBuilder {
	r.sidecarContainerName = "workload"
	r.sidecarImageRepository = "image"
	r.sidecarImageTag = "1.0"
	r.initContainerName = "customer-init"
	r.podAnnotations = map[string]string{}
	return r
}

func (r *SidecarPodFixtureBuilder) SetConditionStatus(value v1.ConditionStatus) *SidecarPodFixtureBuilder {
	r.conditionStatus = value
	return r
}

func (r *SidecarPodFixtureBuilder) SetDeletionTimestamp(value time.Time) *SidecarPodFixtureBuilder {
	r.deletionTimestamp = &metav1.Time{Time: value}
	r.finalizers = []string{"istios.operator.kyma-project.io/test-mock"}
	return r
}

func (r *SidecarPodFixtureBuilder) SetCpuRequest(value string) *SidecarPodFixtureBuilder {
	r.resources.Requests[v1.ResourceCPU] = resource.MustParse(value)
	return r
}

func (r *SidecarPodFixtureBuilder) SetMemoryRequest(value string) *SidecarPodFixtureBuilder {
	r.resources.Requests[v1.ResourceMemory] = resource.MustParse(value)
	return r
}

func (r *SidecarPodFixtureBuilder) SetCpuLimit(value string) *SidecarPodFixtureBuilder {
	r.resources.Limits[v1.ResourceCPU] = resource.MustParse(value)
	return r
}

func (r *SidecarPodFixtureBuilder) SetMemoryLimit(value string) *SidecarPodFixtureBuilder {
	r.resources.Limits[v1.ResourceMemory] = resource.MustParse(value)
	return r
}

func (r *SidecarPodFixtureBuilder) Build() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.name,
			Namespace: r.namespace,
			OwnerReferences: []metav1.OwnerReference{
				r.ownerReference,
			},
			Annotations:       r.podAnnotations,
			Labels:            r.podLabels,
			DeletionTimestamp: r.deletionTimestamp,
			Finalizers:        r.finalizers,
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
					Name:      r.sidecarContainerName,
					Image:     fmt.Sprintf(`%s:%s`, r.sidecarImageRepository, r.sidecarImageTag),
					Resources: r.resources,
				},
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

func FixPodWithOnlySidecar(name, namespace string) *v1.Pod {
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
					Name:      "istio-proxy",
					Image:     "istio/istio-proxy:1.0.0",
					Resources: DefaultSidecarResources,
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
