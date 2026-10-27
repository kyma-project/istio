package evaluation

const (
	// InjectionLabel is the namespace label that enables or disables Istio
	// sidecar injection.
	InjectionLabel = "istio-injection"
	// InjectEnabled is the InjectionLabel value that selects a namespace for
	// injection.
	InjectEnabled = "enabled"
	// InjectDisabled is the InjectionLabel value that excludes a namespace from
	// injection.
	InjectDisabled = "disabled"
	// InjectOverride is the per-workload injection override key. Istio honors it
	// as both a pod label and a pod annotation, so the pod template is checked
	// for either form.
	InjectOverride = "sidecar.istio.io/inject"
)

// ShouldInject reports whether a workload should carry an Istio sidecar, given
// its namespace labels and its pod-template labels and annotations. It mirrors
// Istio's injection policy under Kyma's webhook configuration
// (enableNamespacesByDefault: false, default revision):
//
//   - The per-workload override sidecar.istio.io/inject — accepted as either a
//     pod-template label or annotation — overrides the namespace: "false" always
//     opts out; "true" opts in unless the namespace is istio-injection=disabled.
//     If the label and annotation disagree, label wins (annotations are deprecated,
//     and the actual state will match the label)
//   - Otherwise the namespace label decides: istio-injection=enabled injects;
//     a disabled or unlabeled namespace does not.
func ShouldInject(nsLabels, templateLabels, templateAnnotations map[string]string) bool {
	nsDisabled := nsLabels[InjectionLabel] == InjectDisabled

	override, set := injectOverride(templateLabels, templateAnnotations)
	if set {
		if !override {
			return false
		}
		return !nsDisabled
	}

	return nsLabels[InjectionLabel] == InjectEnabled
}

// injectOverride reads the sidecar.istio.io/inject override from the pod
// template's labels and annotations. The second return is false when neither is set.
// When both are present and disagree, the label wins: the
// annotation form is deprecated, and the actual injected state tracks the label.
func injectOverride(templateLabels, templateAnnotations map[string]string) (inject, set bool) {
	if lv, lok := templateLabels[InjectOverride]; lok {
		return lv == "true", true
	}
	if av, aok := templateAnnotations[InjectOverride]; aok {
		return av == "true", true
	}
	return false, false
}
