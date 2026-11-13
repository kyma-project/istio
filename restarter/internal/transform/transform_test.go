package transform

import (
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func objectWithManagedFieldsAndOwnerRefs(name string, pod bool) metav1.Object {
	om := metav1.ObjectMeta{
		Name:        name,
		Namespace:   "ns",
		Labels:      map[string]string{"app": "x"},
		Annotations: map[string]string{"a": "b"},
		ManagedFields: []metav1.ManagedFieldsEntry{
			{Manager: "m"},
		},
		OwnerReferences: []metav1.OwnerReference{
			{Name: "owner", APIVersion: "v1", Kind: "Deployment", UID: "uid"},
		},
		Finalizers: []string{"protect.me"},
	}
	if pod {
		return &corev1.Pod{ObjectMeta: om}
	}
	return &appsv1.Deployment{ObjectMeta: om}
}

func TestStripForRestart(t *testing.T) {
	tests := []struct {
		name        string
		in          any
		passthrough bool
	}{
		{name: "deployment", in: objectWithManagedFieldsAndOwnerRefs("dep", false)},
		{name: "pod", in: objectWithManagedFieldsAndOwnerRefs("pod", true)},
		{name: "non-object passthrough", in: "not-a-k8s-object", passthrough: true},
	}

	strip := StripForRestart()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := strip(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.passthrough {
				if out != tt.in {
					t.Errorf("non-object should pass through unchanged, got %v", out)
				}
				return
			}

			acc, err := meta.Accessor(out)
			if err != nil {
				t.Fatalf("output not a metav1.Object: %v", err)
			}
			if len(acc.GetManagedFields()) != 0 {
				t.Errorf("managedFields not stripped: %v", acc.GetManagedFields())
			}
			if len(acc.GetOwnerReferences()) != 0 {
				t.Errorf("ownerReferences not stripped: %v", acc.GetOwnerReferences())
			}
			if len(acc.GetFinalizers()) == 0 {
				t.Errorf("finalizers must be retained, got none")
			}
			if acc.GetName() == "" || acc.GetNamespace() == "" {
				t.Errorf("name/namespace must be retained, got name=%q ns=%q", acc.GetName(), acc.GetNamespace())
			}
			if acc.GetLabels()["app"] != "x" {
				t.Errorf("labels not retained: %v", acc.GetLabels())
			}
			if acc.GetAnnotations()["a"] != "b" {
				t.Errorf("annotations not retained: %v", acc.GetAnnotations())
			}
		})
	}
}

func TestStripForRestartKeepOwnerRefs(t *testing.T) {
	tests := []struct {
		name        string
		in          any
		passthrough bool
	}{
		{name: "deployment", in: objectWithManagedFieldsAndOwnerRefs("dep", false)},
		{name: "pod", in: objectWithManagedFieldsAndOwnerRefs("pod", true)},
		{name: "non-object passthrough", in: "not-a-k8s-object", passthrough: true},
	}

	strip := StripForRestartKeepOwnerRefs()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := strip(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.passthrough {
				if out != tt.in {
					t.Errorf("non-object should pass through unchanged, got %v", out)
				}
				return
			}

			acc, err := meta.Accessor(out)
			if err != nil {
				t.Fatalf("output not a metav1.Object: %v", err)
			}
			if len(acc.GetManagedFields()) != 0 {
				t.Errorf("managedFields not stripped: %v", acc.GetManagedFields())
			}
			if len(acc.GetOwnerReferences()) != 1 {
				t.Errorf("ownerReferences must be retained, got %v", acc.GetOwnerReferences())
			}
			if len(acc.GetFinalizers()) == 0 {
				t.Errorf("finalizers must be retained, got none")
			}
			if acc.GetName() == "" || acc.GetNamespace() == "" {
				t.Errorf("name/namespace must be retained, got name=%q ns=%q", acc.GetName(), acc.GetNamespace())
			}
			if acc.GetLabels()["app"] != "x" {
				t.Errorf("labels not retained: %v", acc.GetLabels())
			}
			if acc.GetAnnotations()["a"] != "b" {
				t.Errorf("annotations not retained: %v", acc.GetAnnotations())
			}
		})
	}
}

func fullPod() *corev1.Pod {
	trueBool := true
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "pod",
			Namespace:   "ns",
			Labels:      map[string]string{"app": "x"},
			Annotations: map[string]string{"sidecar.istio.io/status": "{}"},
			ManagedFields: []metav1.ManagedFieldsEntry{
				{Manager: "m"},
			},
			OwnerReferences: []metav1.OwnerReference{
				{Name: "owner", APIVersion: "apps/v1", Kind: "ReplicaSet", UID: "uid"},
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
			Volumes: []corev1.Volume{
				{Name: "vol-1"},
			},
			InitContainers: []corev1.Container{
				{
					Name:    "istio-init",
					Image:   "init:1",
					Command: []string{"/bin/sh"},
					Args:    []string{"-c", "iptables"},
					Env: []corev1.EnvVar{
						{Name: "SOME_VAR", Value: "some-value"},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "vol-1", MountPath: "/mnt"},
					},
					Ports: []corev1.ContainerPort{
						{ContainerPort: 15000},
					},
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot: &trueBool,
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:    "app",
					Image:   "app:1",
					Command: []string{"/bin/app"},
					Env: []corev1.EnvVar{
						{Name: "APP_ENV", Value: "prod"},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "vol-1", MountPath: "/data"},
					},
				},
				{
					Name:    "istio-proxy",
					Image:   "proxy:1",
					Command: []string{"/usr/local/bin/pilot-agent"},
					Args:    []string{"proxy", "sidecar"},
					Env: []corev1.EnvVar{
						{Name: "PILOT_ADDRESS", Value: "istiod:15012"},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "vol-1", MountPath: "/etc/certs"},
					},
					Ports: []corev1.ContainerPort{
						{ContainerPort: 15090},
					},
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot: &trueBool,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "istio-proxy", Ready: true},
			},
		},
	}
}

func TestStripForPod(t *testing.T) {
	tests := []struct {
		name        string
		in          any
		passthrough bool
	}{
		{name: "pod trimmed to minimal fields", in: fullPod()},
		{
			name: "non-pod passthrough",
			in: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dep",
					Namespace: "ns",
					ManagedFields: []metav1.ManagedFieldsEntry{
						{Manager: "m"},
					},
				},
			},
			passthrough: true,
		},
	}

	strip := StripForPod()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := strip(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.passthrough {
				if out != tt.in {
					t.Errorf("non-pod should pass through unchanged")
				}
				dep := tt.in.(*appsv1.Deployment)
				if len(dep.ManagedFields) != 1 {
					t.Errorf("non-pod should be unmodified, managedFields = %v", dep.ManagedFields)
				}
				return
			}

			pod := tt.in.(*corev1.Pod)
			gotPod, ok := out.(*corev1.Pod)
			if !ok {
				t.Fatalf("output is not a *corev1.Pod: %T", out)
			}
			if gotPod != pod {
				t.Errorf("expected the same pod pointer returned")
			}

			if gotPod.ManagedFields != nil {
				t.Errorf("managedFields not stripped: %v", gotPod.ManagedFields)
			}

			emptyStatus := corev1.PodStatus{}
			if !reflect.DeepEqual(gotPod.Status, emptyStatus) {
				t.Errorf("status not zeroed: %+v", gotPod.Status)
			}

			if len(gotPod.Spec.Volumes) != 0 {
				t.Errorf("volumes not dropped: %v", gotPod.Spec.Volumes)
			}
			if gotPod.Spec.NodeName != "" {
				t.Errorf("nodeName not dropped: %q", gotPod.Spec.NodeName)
			}

			if len(gotPod.Spec.InitContainers) != 1 {
				t.Fatalf("initContainers not retained, got %d", len(gotPod.Spec.InitContainers))
			}
			initC := gotPod.Spec.InitContainers[0]
			if initC.Name != "istio-init" {
				t.Errorf("init container name = %q, want %q", initC.Name, "istio-init")
			}
			if initC.Image != "init:1" {
				t.Errorf("init container image = %q, want %q", initC.Image, "init:1")
			}
			if len(initC.Env) != 0 {
				t.Errorf("init container Env not dropped: %v", initC.Env)
			}
			if len(initC.VolumeMounts) != 0 {
				t.Errorf("init container VolumeMounts not dropped: %v", initC.VolumeMounts)
			}
			if len(initC.Ports) != 0 {
				t.Errorf("init container Ports not dropped: %v", initC.Ports)
			}
			if initC.SecurityContext != nil {
				t.Errorf("init container SecurityContext not dropped: %v", initC.SecurityContext)
			}
			if len(initC.Command) != 0 {
				t.Errorf("init container Command not dropped: %v", initC.Command)
			}
			if len(initC.Args) != 0 {
				t.Errorf("init container Args not dropped: %v", initC.Args)
			}

			if len(gotPod.Spec.Containers) != 2 {
				t.Fatalf("containers not retained, got %d", len(gotPod.Spec.Containers))
			}
			var proxy *corev1.Container
			for i := range gotPod.Spec.Containers {
				if gotPod.Spec.Containers[i].Name == "istio-proxy" {
					proxy = &gotPod.Spec.Containers[i]
				}
			}
			if proxy == nil {
				t.Fatal("istio-proxy container not retained")
			}
			if proxy.Image != "proxy:1" {
				t.Errorf("istio-proxy image = %q, want %q", proxy.Image, "proxy:1")
			}
			if proxy.Resources.Requests.Cpu().String() != "100m" {
				t.Errorf("istio-proxy cpu request = %q, want %q", proxy.Resources.Requests.Cpu().String(), "100m")
			}
			if proxy.Resources.Requests.Memory().String() != "128Mi" {
				t.Errorf("istio-proxy memory request = %q, want %q", proxy.Resources.Requests.Memory().String(), "128Mi")
			}
			if proxy.Resources.Limits.Cpu().String() != "200m" {
				t.Errorf("istio-proxy cpu limit = %q, want %q", proxy.Resources.Limits.Cpu().String(), "200m")
			}
			if proxy.Resources.Limits.Memory().String() != "256Mi" {
				t.Errorf("istio-proxy memory limit = %q, want %q", proxy.Resources.Limits.Memory().String(), "256Mi")
			}
			if len(proxy.Env) != 0 {
				t.Errorf("istio-proxy Env not dropped: %v", proxy.Env)
			}
			if len(proxy.VolumeMounts) != 0 {
				t.Errorf("istio-proxy VolumeMounts not dropped: %v", proxy.VolumeMounts)
			}
			if len(proxy.Ports) != 0 {
				t.Errorf("istio-proxy Ports not dropped: %v", proxy.Ports)
			}
			if proxy.SecurityContext != nil {
				t.Errorf("istio-proxy SecurityContext not dropped: %v", proxy.SecurityContext)
			}
			if len(proxy.Command) != 0 {
				t.Errorf("istio-proxy Command not dropped: %v", proxy.Command)
			}
			if len(proxy.Args) != 0 {
				t.Errorf("istio-proxy Args not dropped: %v", proxy.Args)
			}

			var appC *corev1.Container
			for i := range gotPod.Spec.Containers {
				if gotPod.Spec.Containers[i].Name == "app" {
					appC = &gotPod.Spec.Containers[i]
				}
			}
			if appC == nil {
				t.Fatal("app container not retained")
			}
			if appC.Image != "app:1" {
				t.Errorf("app container image = %q, want %q", appC.Image, "app:1")
			}
			if len(appC.Env) != 0 {
				t.Errorf("app container Env not dropped: %v", appC.Env)
			}
			if len(appC.VolumeMounts) != 0 {
				t.Errorf("app container VolumeMounts not dropped: %v", appC.VolumeMounts)
			}
			if len(appC.Command) != 0 {
				t.Errorf("app container Command not dropped: %v", appC.Command)
			}

			if len(gotPod.OwnerReferences) != 1 || gotPod.OwnerReferences[0].Name != "owner" {
				t.Errorf("ownerReferences not retained: %v", gotPod.OwnerReferences)
			}
			if gotPod.Annotations["sidecar.istio.io/status"] != "{}" {
				t.Errorf("annotations not retained: %v", gotPod.Annotations)
			}
			if gotPod.Labels["app"] != "x" {
				t.Errorf("labels not retained: %v", gotPod.Labels)
			}
		})
	}
}
