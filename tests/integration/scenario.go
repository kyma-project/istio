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
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" has condition with reason "([^"]*)" of type "([^"]*)" and status "([^"]*)"$`, steps.IstioCRInNamespaceHasStatusCondition)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" has description "([^"]*)"$`, steps.IstioCRInNamespaceHasDescription)
	ctx.Step(`^Template value "([^"]*)" is set to "([^"]*)"$`, t.SetTemplateValue)
	ctx.Step(`^Istio CR "([^"]*)" from "([^"]*)" is applied in namespace "([^"]*)"$`, t.IstioCRIsAppliedInNamespace)
	ctx.Step(`^Istio CR "([^"]*)" from "([^"]*)" is updated in namespace "([^"]*)"$`, t.IstioCRIsUpdatedInNamespace)
	ctx.Step(`^Namespace "([^"]*)" is "([^"]*)"$`, steps.NamespaceIsPresent)
	ctx.Step(`^Namespace "([^"]*)" is created$`, steps.NamespaceIsCreated)
	ctx.Step(`^Namespace "([^"]*)" has "([^"]*)" label and "([^"]*)" annotation`, steps.NamespaceHasLabelAndAnnotation)
	ctx.Step(`^Istio CRDs "([^"]*)" be present on cluster$`, steps.IstioCRDsBePresentOnCluster)
	ctx.Step(`^"([^"]*)" has "([^"]*)" set to cpu - "([^"]*)" and memory - "([^"]*)"$`, steps.IstioComponentHasResourcesSetToCpuAndMemory)
	ctx.Step(`^Pod of deployment "([^"]*)" in namespace "([^"]*)" has container "([^"]*)" with resource "([^"]*)" set to cpu - "([^"]*)" and memory - "([^"]*)"$`, steps.DeploymentHasPodWithContainerResourcesSetToCpuAndMemory)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is deleted$`, steps.ResourceInNamespaceIsDeleted)
	ctx.Step(`^"([^"]*)" "([^"]*)" is deleted$`, steps.ClusterResourceIsDeleted)
	ctx.Step(`^"([^"]*)" is not present on cluster$`, steps.ResourceNotPresent)
	ctx.Step(`^Istio injection is "([^"]*)" in namespace "([^"]*)"$`, steps.SetIstioInjection)
	ctx.Step(`^Application "([^"]*)" in namespace "([^"]*)" has proxy with "([^"]*)" set to cpu - "([^"]*)" and memory - "([^"]*)"$`, steps.ApplicationHasProxyResourcesSetToCpuAndMemory)
	ctx.Step(`^Application pod "([^"]*)" in namespace "([^"]*)" has Istio proxy "([^"]*)"$`, steps.ApplicationPodShouldHaveIstioProxy)
	ctx.Step(`^Destination rule "([^"]*)" in namespace "([^"]*)" with host "([^"]*)" exists$`, steps.CreateDestinationRule)
	ctx.Step(`^Istio is manually uninstalled$`, steps.UninstallIstio)
	ctx.Step(`^Httpbin application "([^"]*)" deployment is created in namespace "([^"]*)"$`, steps.CreateHttpbinApplication)
	ctx.Step(`^Httpbin application "([^"]*)" deployment is created in namespace "([^"]*)" with service port "([^"]*)"$`, steps.CreateHttpbinApplicationWithServicePort)
	ctx.Step(`^Nginx application "([^"]*)" deployment is created in namespace "([^"]*)" with forward to "([^"]*)" and service port 80$`, steps.CreateNginxApplication)
	ctx.Step(`^Istio gateway "([^"]*)" is configured in namespace "([^"]*)"$`, steps.CreateIstioGateway)
	ctx.Step(`^Virtual service "([^"]*)" exposing service "([^"]*)" by gateway "([^"]*)" is configured in namespace "([^"]*)"$`, steps.CreateVirtualService)
	ctx.Step(`^Virtual service "([^"]*)" exposing service "([^"]*)" with port "([^"]*)" by gateway "([^"]*)" is configured in namespace "([^"]*)"$`, steps.CreateVirtualServiceWithPort)
	ctx.Step(`^Request with header "([^"]*)" with value "([^"]*)" sent to httpbin should return "([^"]*)" with value "([^"]*)"$`, steps.ValidateHeader)
	ctx.Step(`^Request to path "([^"]*)" should return "([^"]*)" with value "([^"]*)" in body$`, steps.ValidateHeaderInBody)
	ctx.Step(`^Request to path "([^"]*)" should have response code "([^"]*)"$`, steps.ValidateResponseStatusCode)
	ctx.Step(`^Request with header "([^"]*)" with value "([^"]*)" to path "([^"]*)" should have response code "([^"]*)"$`, steps.ValidateResponseCodeForRequestWithHeader)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is "([^"]*)"`, steps.ResourceIsPresent)
	ctx.Step(`^Request sent to exposed httpbin, should contain public client IP in "([^"]*)" header$`, steps.ValidatePublicClientIpInHeader)
	ctx.Step(`^Access logging is enabled for the mesh using "([^"]*)" provider$`, steps.EnableAccessLogging)
	ctx.Step(`^Log of container "([^"]*)" in deployment "([^"]*)" in namespace "([^"]*)" contains "([^"]*)"$`, steps.ContainerLogContainsString)
	ctx.Step(`^Tracing is enabled for the mesh using provider "([^"]*)"$`, steps.EnableTracing)
	ctx.Step(`^Service is created for the otel collector "([^"]*)" in namespace "([^"]*)"$`, steps.CreateOpenTelemetryService)
	ctx.Step(`^OTEL Collector mock "([^"]*)" deployment is created in namespace "([^"]*)"$`, steps.CreateTelemetryCollectorMock)
	ctx.Step(`^Ext-authz application "([^"]*)" deployment is created in namespace "([^"]*)"$`, steps.CreateExtAuthzApplication)
	ctx.Step(`^Authorization policy "([^"]*)" in namespace "([^"]*)" with app selector "([^"]*)" is using extension provider "([^"]*)" for operation "([^"]*)"$`, steps.CreateAuthorizationPolicyExtAuthz)
	ctx.Step(`Environment variable "([^"]*)" on Deployment "([^"]*)" in namespace "([^"]*)" is present and has value "([^"]*)`, steps.VerifyEnvVariableOnDeployment)
	ctx.Step(`Environment variable "([^"]*)" on Deployment "([^"]*)" in namespace "([^"]*)" is not present`, steps.VerifyEnvVariableIsNotOnDeployment)
}

func upgradeInitScenario(ctx *godog.ScenarioContext) {
	ctx.After(verifyIfControllerHasBeenRestarted)
	ctx.After(testObjectsTearDown)
	ctx.After(istioCrTearDown)

	t := steps.TemplatedIstioCr{}

	ctx.Step(`^"([^"]*)" is not present on cluster$`, steps.ResourceNotPresent)
	ctx.Step(`^Istio CRD is installed$`, steps.IstioCRDIsInstalled)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is ready$`, steps.ResourceIsReady)
	ctx.Step(`^Istio CR "([^"]*)" from "([^"]*)" is applied in namespace "([^"]*)"$`, t.IstioCRIsAppliedInNamespace)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" has status "([^"]*)"$`, steps.IstioCRInNamespaceHasStatus)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" status update happened in the last 20 seconds$`, steps.IstioCrStatusUpdateHappened)
	ctx.Step(`^Istio injection is "([^"]*)" in namespace "([^"]*)"$`, steps.SetIstioInjection)
	ctx.Step(`^Httpbin application "([^"]*)" deployment is created in namespace "([^"]*)"$`, steps.CreateHttpbinApplication)
	ctx.Step(`^Application pod "([^"]*)" in namespace "([^"]*)" has Istio proxy "([^"]*)"$`, steps.ApplicationPodShouldHaveIstioProxy)
	ctx.Step(`^Istio controller has been upgraded with "([^"]*)" manifest and should "([^"]*)"$`, steps.DeployIstioOperator)
	ctx.Step(`^Application "([^"]*)" in namespace "([^"]*)" has required version of proxy$`, steps.ApplicationPodShouldHaveIstioProxyInRequiredVersion)
	ctx.Step(`^Container "([^"]*)" of "([^"]*)" "([^"]*)" in namespace "([^"]*)" has required version$`, steps.IstioResourceContainerHasRequiredVersion)
}
