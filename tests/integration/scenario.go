package integration

import (
	"github.com/cucumber/godog"
	"github.com/kyma-project/istio/operator/tests/integration/steps"
)

func initScenario(ctx *godog.ScenarioContext) {

	ctx.After(testObjectsTearDown)
	ctx.After(istioCrTearDown)

	t := steps.TemplatedIstioCr{}
	ctx.Step(`^Evaluated cluster size is "([^"]*)"$`, steps.EvaluatedClusterSizeIs)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is ready`, steps.ResourceIsReady)
	ctx.Step(`^Istio CRD is installed$`, steps.IstioCRDIsInstalled)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" has status "([^"]*)"$`, steps.IstioCRInNamespaceHasStatus)
	ctx.Step(`^Template value "([^"]*)" is set to "([^"]*)"$`, t.SetTemplateValue)
	ctx.Step(`^Istio CR "([^"]*)" is applied in namespace "([^"]*)"$`, t.IstioCRIsAppliedInNamespace)
	ctx.Step(`^Istio CR "([^"]*)" is updated in namespace "([^"]*)"$`, t.IstioCrIsUpdatedInNamespace)
	ctx.Step(`^Namespace "([^"]*)" is "([^"]*)"$`, steps.NamespaceIsPresent)
	ctx.Step(`^Namespace "([^"]*)" has "([^"]*)" label and "([^"]*)" annotation`, steps.NamespaceHasLabelAndAnnotation)
	ctx.Step(`^Istio CRDs "([^"]*)" be present on cluster$`, steps.IstioCRDsBePresentOnCluster)
	ctx.Step(`^"([^"]*)" has "([^"]*)" set to cpu - "([^"]*)" and memory - "([^"]*)"$`, steps.IstioComponentHasResourcesSetToCpuAndMemory)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is deleted$`, steps.ResourceInNamespaceIsDeleted)
	ctx.Step(`^"([^"]*)" is not present on cluster$`, steps.ResourceNotPresent)
	ctx.Step(`^Istio injection is enabled in namespace "([^"]*)"$`, steps.EnableIstioInjection)
	ctx.Step(`^Application "([^"]*)" is running in namespace "([^"]*)"$`, steps.CreateApplicationDeployment)
	ctx.Step(`^Application "([^"]*)" in namespace "([^"]*)" has proxy with "([^"]*)" set to cpu - "([^"]*)" and memory - "([^"]*)"$`, steps.ApplicationHasProxyResourcesSetToCpuAndMemory)
	ctx.Step(`^Application pod "([^"]*)" in namespace "([^"]*)" has Istio proxy "([^"]*)"$`, steps.ApplicationPodShouldHaveIstioProxy)
	ctx.Step(`^Destination rule "([^"]*)" in namespace "([^"]*)" with host "([^"]*)" exists$`, steps.CreateDestinationRule)
}
