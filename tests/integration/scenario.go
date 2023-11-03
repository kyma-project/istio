package integration

import (
	"github.com/cucumber/godog"
	"github.com/kyma-project/istio/operator/tests/integration/steps"
)

func initScenario(ctx *godog.ScenarioContext) {

	ctx.After(verifyIfControllerHasBeenRestarted)
	ctx.After(testObjectsTearDown)
	ctx.After(istioCrTearDown)

	t := steps.TemplatedIstioCr{}
	ctx.Step(`^Evaluated cluster size is "([^"]*)"$`, steps.EvaluatedClusterSizeIs)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is ready`, steps.ResourceIsReady)
	ctx.Step(`^Istio CRD is installed$`, steps.IstioCRDIsInstalled)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" has status "([^"]*)"$`, steps.IstioCRInNamespaceHasStatus)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" has description "([^"]*)"$`, steps.IstioCRInNamespaceHasDescription)
	ctx.Step(`^Template value "([^"]*)" is set to "([^"]*)"$`, t.SetTemplateValue)
	ctx.Step(`^Istio CR "([^"]*)" is applied in namespace "([^"]*)"$`, t.IstioCRIsAppliedInNamespace)
	ctx.Step(`^Istio CR "([^"]*)" is updated in namespace "([^"]*)"$`, t.IstioCRIsUpdatedInNamespace)
	ctx.Step(`^Namespace "([^"]*)" is "([^"]*)"$`, steps.NamespaceIsPresent)
	ctx.Step(`^Namespace "([^"]*)" is created$`, steps.NamespaceIsCreated)
	ctx.Step(`^Namespace "([^"]*)" has "([^"]*)" label and "([^"]*)" annotation`, steps.NamespaceHasLabelAndAnnotation)
	ctx.Step(`^Istio CRDs "([^"]*)" be present on cluster$`, steps.IstioCRDsBePresentOnCluster)
	ctx.Step(`^"([^"]*)" has "([^"]*)" set to cpu - "([^"]*)" and memory - "([^"]*)"$`, steps.IstioComponentHasResourcesSetToCpuAndMemory)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is deleted$`, steps.ResourceInNamespaceIsDeleted)
	ctx.Step(`^"([^"]*)" "([^"]*)" is deleted$`, steps.ClusterResourceIsDeleted)
	ctx.Step(`^"([^"]*)" is not present on cluster$`, steps.ResourceNotPresent)
	ctx.Step(`^Istio injection is "([^"]*)" in namespace "([^"]*)"$`, steps.SetIstioInjection)
	ctx.Step(`^Application "([^"]*)" is running in namespace "([^"]*)"$`, steps.CreateApplicationDeployment)
	ctx.Step(`^Application "([^"]*)" in namespace "([^"]*)" has proxy with "([^"]*)" set to cpu - "([^"]*)" and memory - "([^"]*)"$`, steps.ApplicationHasProxyResourcesSetToCpuAndMemory)
	ctx.Step(`^Application pod "([^"]*)" in namespace "([^"]*)" has Istio proxy "([^"]*)"$`, steps.ApplicationPodShouldHaveIstioProxy)
	ctx.Step(`^Destination rule "([^"]*)" in namespace "([^"]*)" with host "([^"]*)" exists$`, steps.CreateDestinationRule)
	ctx.Step(`^Istio is manually uninstalled$`, steps.UninstallIstio)
	ctx.Step(`^Istio "([^"]*)" service has annotation "([^"]*)" on "([^"]*)" cluster$`, steps.IstioServiceHasAnnotation)
	ctx.Step(`^Httpbin application "([^"]*)" is running in namespace "([^"]*)"$`, steps.CreateHttpbinApplication)
	ctx.Step(`^Httpbin application "([^"]*)" is running in namespace "([^"]*)" with service port "([^"]*)"$`, steps.CreateHttpbinApplicationWithServicePort)
	ctx.Step(`^Nginx application "([^"]*)" is running in namespace "([^"]*)" with forward to "([^"]*)" and service port 80$`, steps.CreateNginxApplication)
	ctx.Step(`^Istio gateway "([^"]*)" is configured in namespace "([^"]*)"$`, steps.CreateIstioGateway)
	ctx.Step(`^Virtual service "([^"]*)" exposing service "([^"]*)" by gateway "([^"]*)" is configured in namespace "([^"]*)"$`, steps.CreateVirtualService)
	ctx.Step(`^Virtual service "([^"]*)" exposing service "([^"]*)" with port "([^"]*)" by gateway "([^"]*)" is configured in namespace "([^"]*)"$`, steps.CreateVirtualServiceWithPort)
	ctx.Step(`^Request with header X-Forwarded-For with value "([^"]*)" sent to httpbin should return X-Envoy-External-Address with value "([^"]*)"$`, steps.ValidateHeader)
	ctx.Step(`^Request with header "([^"]*)" with value "([^"]*)" sent to httpbin should return "([^"]*)" with value "([^"]*)"$`, steps.ValidateHeader)
	ctx.Step(`^Request to path "([^"]*)" should return "([^"]*)" with value "([^"]*)" in body$`, steps.ValidateHeaderInBody)
	ctx.Step(`^Request to path "([^"]*)" should have response code "([^"]*)"$`, steps.ValidateResponseStatusCode)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is "([^"]*)"`, steps.ResourceIsPresent)
	ctx.Step(`^Request sent to exposed httpbin, should contain public client IP in "([^"]*)" header$`, steps.ValidatePublicClientIpInHeader)
}

func upgradeInitScenario(ctx *godog.ScenarioContext) {

	ctx.After(verifyIfControllerHasBeenRestarted)
	ctx.After(testObjectsTearDown)
	ctx.After(istioCrTearDown)

	t := steps.TemplatedIstioCr{}

	ctx.Step(`^"([^"]*)" is not present on cluster$`, steps.ResourceNotPresent)
	ctx.Step(`^Istio CRD is installed$`, steps.IstioCRDIsInstalled)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is ready$`, steps.ResourceIsReady)
	ctx.Step(`^Istio CR "([^"]*)" is applied in namespace "([^"]*)"$`, t.IstioCRIsAppliedInNamespace)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" has status "([^"]*)"$`, steps.IstioCRInNamespaceHasStatus)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" status update happened in the last 20 seconds$`, steps.IstioCrStatusUpdateHappened)
	ctx.Step(`^Istio injection is "([^"]*)" in namespace "([^"]*)"$`, steps.SetIstioInjection)
	ctx.Step(`^Application "([^"]*)" is running in namespace "([^"]*)"$`, steps.CreateApplicationDeployment)
	ctx.Step(`^Application pod "([^"]*)" in namespace "([^"]*)" has Istio proxy "([^"]*)"$`, steps.ApplicationPodShouldHaveIstioProxy)
	ctx.Step(`^Istio controller has been upgraded to the new version$`, steps.DeployIstioOperatorFromLocalManifest)
	ctx.Step(`^Application "([^"]*)" in namespace "([^"]*)" has required version of proxy$`, steps.ApplicationPodShouldHaveIstioProxyInRequiredVersion)
	ctx.Step(`^Container "([^"]*)" of "([^"]*)" "([^"]*)" in namespace "([^"]*)" has required version$`, steps.IstioResourceContainerHasRequiredVersion)
}
