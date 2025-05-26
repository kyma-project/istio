package labels

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

const (
	LastAppliedConfiguration string = "operator.kyma-project.io/lastAppliedConfiguration"
	ModuleLabelKey           string = "kyma-project.io/module"
	ModuleLabelValue         string = "istio"
)

func SetModuleLabels(labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	defaultLabels := map[string]string{
		ModuleLabelKey: ModuleLabelValue,
	}
	for k, v := range defaultLabels {
		labels[k] = v
	}
	return labels
}

func HasModuleLabels(item unstructured.Unstructured) bool {
	val, exists := item.GetLabels()[ModuleLabelKey]
	return exists && val == ModuleLabelValue
}
