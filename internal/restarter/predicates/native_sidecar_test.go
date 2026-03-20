package predicates

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNativeSidecarPredicateMatches(t *testing.T) {
	tests := []struct {
		name            string
		isInitContainer bool
		annotations     map[string]string
		expectedResult  bool
	}{
		// container: regular, annotation: not set
		{
			name:            "should evaluate to true, when proxy is a regular container and nativeSidecar annotation is not set",
			isInitContainer: false,
			annotations:     map[string]string{},
			expectedResult:  true,
		},
		// container: regular, annotation: false
		{
			name:            "should evaluate to false, when proxy is a regular container and nativeSidecar annotation is set to false",
			isInitContainer: false,
			annotations:     map[string]string{nativeSidecarAnnotation: "false"},
			expectedResult:  false,
		},
		// container: regular, annotation: true
		{
			name:            "should evaluate to true, when proxy is a regular container and nativeSidecar annotation is set to true",
			isInitContainer: false,
			annotations:     map[string]string{nativeSidecarAnnotation: "true"},
			expectedResult:  true,
		},
		// container: initContainer, annotation: not set
		{
			name:            "should evaluate to false, when proxy is an init container and nativeSidecar annotation is not set",
			isInitContainer: true,
			annotations:     map[string]string{},
			expectedResult:  false,
		},
		// container: initContainer, annotation: false
		{
			name:            "should evaluate to true, when proxy is an init container and nativeSidecar annotation is set to false",
			isInitContainer: true,
			annotations:     map[string]string{nativeSidecarAnnotation: "false"},
			expectedResult:  true,
		},
		// container: initContainer, annotation: true
		{
			name:            "should evaluate to false, when proxy is an init container and nativeSidecar annotation is set to true",
			isInitContainer: true,
			annotations:     map[string]string{nativeSidecarAnnotation: "true"},
			expectedResult:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			predicate := NewNativeSidecarRestartPredicate()
			pod := createIstioInjectedPod(tt.isInitContainer, tt.annotations)
			require.Equal(t, tt.expectedResult, predicate.Matches(pod))
		})
	}
}

func createIstioInjectedPod(isInitContainer bool, annotations map[string]string) v1.Pod {
	if isInitContainer {
		return v1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: annotations}, Spec: v1.PodSpec{InitContainers: []v1.Container{{Name: "istio-proxy"}}}}
	}
	return v1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: annotations}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "istio-proxy"}}}}
}
