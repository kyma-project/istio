package labels

func SetModuleLabels(labels map[string]string) map[string]string {

	if labels == nil {
		labels = make(map[string]string)
	}
	defaultLabels := map[string]string{
		"kyma-project.io/module": "istio",
	}
	for k, v := range defaultLabels {
		labels[k] = v
	}
	return labels
}
