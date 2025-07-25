package integration

import (
	"github.com/cucumber/godog"
	"github.com/kyma-project/istio/operator/tests/integration/steps"
	"github.com/kyma-project/istio/operator/tests/integration/testsupport"
	"log"
)

func initScenario(ctx *godog.ScenarioContext) {
	ctx.After(getFailedTestIstioCRStatus)
	ctx.After(testObjectsTearDown)
	ctx.After(istioCrTearDown)
	t := steps.TemplatedIstioCr{}

	err := testsupport.LoadEnvironmentVariables()
	if err != nil {
		panic("Failed to set environment variables: " + err.Error())
	}
	log.Printf("initializing tests with Istio Module version: %s", testsupport.GetOperatorVersion())

	// This structure holds all Tester instances
	// every test scenario should have an own instance to allow parallel execution
	zd := steps.ZeroDowntimeTestRunner{}
	ctx.After(zd.CleanZeroDowntimeTests)

	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is "([^"]*)"`, steps.ResourceIsPresent)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is deleted$`, steps.ResourceInNamespaceIsDeleted)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is ready$`, steps.ResourceIsReady)
	ctx.Step(`^"([^"]*)" "([^"]*)" is deleted$`, steps.ClusterResourceIsDeleted)
	ctx.Step(`^"([^"]*)" has "([^"]*)" set to cpu - "([^"]*)" and memory - "([^"]*)"$`, steps.IstioComponentHasResourcesSetToCpuAndMemory)
	ctx.Step(`^"([^"]*)" is not present on cluster$`, steps.ResourceNotPresent)
	ctx.Step(`^Access logging is enabled for the mesh using "([^"]*)" provider$`, steps.EnableAccessLogging)
	ctx.Step(`^Application "([^"]*)" in namespace "([^"]*)" has proxy with "([^"]*)" set to cpu - "([^"]*)" and memory - "([^"]*)"$`, steps.ApplicationHasProxyResourcesSetToCpuAndMemory)
	ctx.Step(`^Application "([^"]*)" in namespace "([^"]*)" has required version of proxy$`, steps.ApplicationPodShouldHaveIstioProxyInRequiredVersion)
	ctx.Step(`^Application pod "([^"]*)" in namespace "([^"]*)" has Istio proxy "([^"]*)"$`, steps.ApplicationPodShouldHaveIstioProxy)
	ctx.Step(`^Authorization policy "([^"]*)" in namespace "([^"]*)" with app selector "([^"]*)" is using extension provider "([^"]*)" for operation "([^"]*)"$`, steps.CreateAuthorizationPolicyExtAuthz)
	ctx.Step(`^Container "([^"]*)" of "([^"]*)" "([^"]*)" in namespace "([^"]*)" has required version$`, steps.IstioResourceContainerHasRequiredVersion)
	ctx.Step(`^Destination rule "([^"]*)" in namespace "([^"]*)" with host "([^"]*)" exists$`, steps.CreateDestinationRule)
	ctx.Step(`^Evaluated cluster size is "([^"]*)"$`, steps.EvaluatedClusterSizeIs)
	ctx.Step(`^Ext-authz application "([^"]*)" deployment is created in namespace "([^"]*)"$`, steps.CreateExtAuthzApplication)
	ctx.Step(`^Httpbin application "([^"]*)" deployment is created in namespace "([^"]*)" with service port "([^"]*)"$`, steps.CreateHttpbinApplicationWithServicePort)
	ctx.Step(`^Httpbin application "([^"]*)" deployment is created in namespace "([^"]*)"$`, steps.CreateHttpbinApplication)
	ctx.Step(`^Istio CR "([^"]*)" from "([^"]*)" is applied in namespace "([^"]*)"$`, t.IstioCRIsAppliedInNamespace)
	ctx.Step(`^Istio CR "([^"]*)" from "([^"]*)" is updated in namespace "([^"]*)"$`, t.IstioCRIsUpdatedInNamespace)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" has condition with reason "([^"]*)" of type "([^"]*)" and status "([^"]*)"$`, steps.IstioCRInNamespaceHasStatusCondition)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" has description "([^"]*)"$`, steps.IstioCRInNamespaceHasDescription)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" contains description "([^"]*)"$`, steps.IstioCRInNamespaceContainsDescription)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" has status "([^"]*)"$`, steps.IstioCRInNamespaceHasStatus)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" status update happened in the last 20 seconds$`, steps.IstioCrStatusUpdateHappened)
	ctx.Step(`^Istio CRD is installed$`, steps.IstioCRDIsInstalled)
	ctx.Step(`^Istio CRDs "([^"]*)" be present on cluster$`, steps.IstioCRDsBePresentOnCluster)
	ctx.Step(`^Istio controller has been upgraded$`, steps.DeployIstioOperator)
	ctx.Step(`^Istio gateway "([^"]*)" is configured in namespace "([^"]*)"$`, steps.CreateIstioGateway)
	ctx.Step(`^Istio injection is "([^"]*)" in namespace "([^"]*)"$`, steps.SetIstioInjection)
	ctx.Step(`^Istio is manually uninstalled$`, steps.UninstallIstio)
	ctx.Step(`^Log of container "([^"]*)" in deployment "([^"]*)" in namespace "([^"]*)" contains "([^"]*)"$`, steps.ContainerLogContainsString)
	ctx.Step(`^Namespace "([^"]*)" has "([^"]*)" label and "([^"]*)" annotation`, steps.NamespaceHasLabelAndAnnotation)
	ctx.Step(`^Namespace "([^"]*)" is "([^"]*)"$`, steps.NamespaceIsPresent)
	ctx.Step(`^Namespace "([^"]*)" is created$`, steps.NamespaceIsCreated)
	ctx.Step(`^Nginx application "([^"]*)" deployment is created in namespace "([^"]*)" with forward to "([^"]*)" and service port 80$`, steps.CreateNginxApplication)
	ctx.Step(`^OTEL Collector mock "([^"]*)" deployment is created in namespace "([^"]*)"$`, steps.CreateTelemetryCollectorMock)
	ctx.Step(`^Pod of deployment "([^"]*)" in namespace "([^"]*)" has container "([^"]*)" with resource "([^"]*)" set to cpu - "([^"]*)" and memory - "([^"]*)"$`, steps.DeploymentHasPodWithContainerResourcesSetToCpuAndMemory)
	ctx.Step(`^Request sent to exposed httpbin with Host header "([^"]*)", should contain public client IP in "([^"]*)" header$`, steps.ValidatePublicClientIpInHeader)
	ctx.Step(`^Request to path "([^"]*)" should have response code "([^"]*)"$`, steps.ValidateResponseStatusCode)
	ctx.Step(`^Request to host "([^"]*)" and path "([^"]*)" should return "([^"]*)" with value "([^"]*)" in body$`, steps.ValidateHeaderInBody)
	ctx.Step(`^Request with Host "([^"]*)" and header "([^"]*)" with value "([^"]*)" to httpbin should return "([^"]*)" with value "([^"]*)"$`, steps.ValidateStatusForHeader)
	ctx.Step(`^Request with Host "([^"]*)" and header "([^"]*)" with value "([^"]*)" to path "([^"]*)" should have response code "([^"]*)"$`, steps.ValidateStatus)
	ctx.Step(`^Request with header "([^"]*)" with value "([^"]*)" to path "([^"]*)" should have response code "([^"]*)"$`, steps.ValidateResponseCodeForRequestWithHeader)
	ctx.Step(`^Service is created for the otel collector "([^"]*)" in namespace "([^"]*)"$`, steps.CreateOpenTelemetryService)
	ctx.Step(`^Template value "([^"]*)" is set to "([^"]*)"$`, t.SetTemplateValue)
	ctx.Step(`^Tracing is enabled for the mesh using provider "([^"]*)"$`, steps.EnableTracing)
	ctx.Step(`^Virtual service "([^"]*)" exposing service "([^"]*)" by gateway "([^"]*)" is configured in namespace "([^"]*)"$`, steps.CreateVirtualService)
	ctx.Step(`^Virtual service "([^"]*)" exposing service "([^"]*)" with port "([^"]*)" by gateway "([^"]*)" is configured in namespace "([^"]*)"$`, steps.CreateVirtualServiceWithPort)
	ctx.Step(`^Httpbin application "([^"]*)" deployment with proxy as a native sidecar is created in namespace "([^"]*)"$`, steps.ApplicationWithInitSidecarCreated)
	ctx.Step(`^Application "([^"]*)" in namespace "([^"]*)" has init container with Istio proxy present$`, steps.ApplicationHasInitContainerWithIstioProxy)
	ctx.Step(`^Application "([^"]*)" in namespace "([^"]*)" has required version of proxy as an init container$`, steps.ApplicationHasRequiredVersionInitContainerWithIstioProxy)
	ctx.Step(`^There are continuous requests to host "([^"]*)" and path "([^"]*)"`, zd.StartZeroDowntimeTest)
	ctx.Step(`^All continuous requests should succeed`, zd.FinishZeroDowntimeTests)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" has proper value in app.kubernetes.io/version label$`, steps.ManagedResourceHasOperatorVersionLabelWithProperValue)
}
