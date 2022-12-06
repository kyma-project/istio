package pods_test

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type SidecarPodFixtureBuilder struct {
	name, namespace, sidecarImageRepository, sidecarImageTag, sidecarContainerName string
	conditionStatus                                                                v1.ConditionStatus
	deletionTimestamp                                                              *metav1.Time
}

func newSidecarPodBuilder() *SidecarPodFixtureBuilder {
	return &SidecarPodFixtureBuilder{
		name:                   "app",
		namespace:              "custom",
		sidecarImageRepository: "istio/proxyv2",
		sidecarImageTag:        "1.10.0",
		conditionStatus:        "True",
		sidecarContainerName:   "istio-proxy",
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
			Annotations:       map[string]string{"sidecar.istio.io/status": "{\"containers\":[\"istio-proxy\"]}"},
			DeletionTimestamp: r.deletionTimestamp,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: v1.PodStatus{
			Phase: "Running",
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
					Name:  "istio-init",
					Image: "istio-init",
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

func fixPodWithoutInitContainer(name, namespace, phase string, annotations map[string]string, labels map[string]string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet"},
			},
			Labels:      labels,
			Annotations: annotations,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: v1.PodStatus{
			Phase: v1.PodPhase(phase),
		},
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{
				{
					Name:  "istio-validation",
					Image: "istio-validation",
				},
			},
			Containers: []v1.Container{
				{
					Name:  name + "-container",
					Image: "image:6.9",
				},
				{
					Name:  "istio-proxy",
					Image: "istio-proxy",
				},
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
