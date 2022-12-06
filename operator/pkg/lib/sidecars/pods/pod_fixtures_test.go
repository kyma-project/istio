package pods_test

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fixPodWithProxyImage(podName, namespace, proxyImageRepository, proxyImageTag string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet"},
			},
			Annotations: map[string]string{"sidecar.istio.io/status": fmt.Sprintf(`{"containers":[%s]}`, proxyImageRepository)},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: v1.PodStatus{
			Phase: "Running",
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
