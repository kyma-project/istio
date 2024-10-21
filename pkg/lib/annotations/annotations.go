package annotations

import (
	"time"
)

const (
	restartAnnotationName = "istio-operator.kyma-project.io/restartedAt"
)

func AddRestartAnnotation(annotations map[string]string) map[string]string {
	if len(annotations) == 0 {
		annotations = map[string]string{}
	}

	annotations[restartAnnotationName] = time.Now().Format(time.RFC3339)
	return annotations
}

func HasRestartAnnotation(annotations map[string]string) bool {
	_, found := annotations[restartAnnotationName]
	return found
}
