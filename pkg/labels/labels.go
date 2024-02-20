package labels

const (
	ModuleLabelKey   = "kyma-project.io/module"
	ModuleLabelValue = "istio"
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
