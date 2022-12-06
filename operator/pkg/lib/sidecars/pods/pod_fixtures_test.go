package pods_test

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func fixPodWithSidecar(podName, namespace, proxyImageRepository, proxyImageTag string) *v1.Pod {
	return fixPodWithSidecarAndConditionStatus(podName, namespace, proxyImageRepository, proxyImageTag, "True")
}

func fixPodWithSidecarAndConditionStatus(podName, namespace, proxyImageRepository, proxyImageTag string, conditionStatus v1.ConditionStatus) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet"},
			},
			Annotations: map[string]string{"sidecar.istio.io/status": "{\"containers\":[\"istio-proxy\"]}"},
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
					Status: conditionStatus,
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
					Name:  "istio-proxy",
					Image: fmt.Sprintf(`%s:%s`, proxyImageRepository, proxyImageTag)},
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

func fixPodWithSidecarWithDeletionTimestamp(podName, namespace, proxyImageRepository, proxyImageTag string) *v1.Pod {
	pod := fixPodWithSidecar(podName, namespace, proxyImageRepository, proxyImageTag)
	pod.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	return pod
}

func fixPodWithSidecarWithSidecarContainerName(podName, namespace, proxyImageRepository, proxyImageTag, sidecarContainerName string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet"},
			},
			Annotations: map[string]string{"sidecar.istio.io/status": "{\"containers\":[\"istio-proxy\"]}"},
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
					Status: "True",
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
					Name:  sidecarContainerName,
					Image: fmt.Sprintf(`%s:%s`, proxyImageRepository, proxyImageTag)},
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
