package pods_test

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type SidecarPodFixtureBuilder struct {
	name, namespace                                               string
	sidecarContainerName, sidecarImageRepository, sidecarImageTag string
	initContainerName                                             string
	podAnnotations                                                map[string]string
	podStatusPhase                                                v1.PodPhase
	conditionStatus                                               v1.ConditionStatus
	deletionTimestamp                                             *metav1.Time
}

func newSidecarPodBuilder() *SidecarPodFixtureBuilder {
	return &SidecarPodFixtureBuilder{
		name:                   "app",
		namespace:              "custom",
		sidecarContainerName:   "istio-proxy",
		sidecarImageRepository: "istio/proxyv2",
		sidecarImageTag:        "1.10.0",
		initContainerName:      "istio-init",
		podAnnotations:         map[string]string{"sidecar.istio.io/status": "{\"containers\":[\"istio-proxy\"]}"},
		podStatusPhase:         "Running",
		conditionStatus:        "True",
	}
}

func (r *SidecarPodFixtureBuilder) setName(value string) *SidecarPodFixtureBuilder {
	r.name = value
	return r
}

func (r *SidecarPodFixtureBuilder) setNamespace(value string) *SidecarPodFixtureBuilder {
	r.namespace = value
	return r
}

func (r *SidecarPodFixtureBuilder) setPodStatusPhase(value v1.PodPhase) *SidecarPodFixtureBuilder {
	r.podStatusPhase = value
	return r
}

func (r *SidecarPodFixtureBuilder) setInitContainer(value string) *SidecarPodFixtureBuilder {
	r.initContainerName = value
	return r
}

func (r *SidecarPodFixtureBuilder) setSidecarImageRepository(value string) *SidecarPodFixtureBuilder {
	r.sidecarImageRepository = value
	return r
}

func (r *SidecarPodFixtureBuilder) setSidecarImageTag(value string) *SidecarPodFixtureBuilder {
	r.sidecarImageTag = value
	return r
}

func (r *SidecarPodFixtureBuilder) setSidecarContainerName(value string) *SidecarPodFixtureBuilder {
	r.sidecarContainerName = value
	return r
}

func (r *SidecarPodFixtureBuilder) disableSidecar() *SidecarPodFixtureBuilder {
	r.sidecarContainerName = "workload"
	r.sidecarImageRepository = "image"
	r.sidecarImageTag = "1.0"
	r.initContainerName = "customer-init"
	r.podAnnotations = map[string]string{}
	return r
}

func (r *SidecarPodFixtureBuilder) setConditionStatus(value v1.ConditionStatus) *SidecarPodFixtureBuilder {
	r.conditionStatus = value
	return r
}

func (r *SidecarPodFixtureBuilder) setDeletionTimestamp(value time.Time) *SidecarPodFixtureBuilder {
	r.deletionTimestamp = &metav1.Time{Time: value}
	return r
}

func (r *SidecarPodFixtureBuilder) build() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.name,
			Namespace: r.namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet"},
			},
			Annotations:       r.podAnnotations,
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
		},
	}
}

func fixPodWithoutSidecar(name, namespace string) *v1.Pod {
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

func fixNamespaceWith(name string, labels map[string]string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}
