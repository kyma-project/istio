// Istio install was identified to be prone to memory leaks.
// As so, we want to separate the dependency istio installation by running it in a separate process.
// This is the file responsible for executing call to istio install
package main

import (
	"os"
	"time"

	istioclient "github.com/kyma-project/istio/operator/internal/reconciliations/istio"

	"istio.io/istio/istioctl/pkg/install/k8sversion"
	istio "istio.io/istio/operator/cmd/mesh"
	"istio.io/istio/pkg/kube"
	"k8s.io/client-go/rest"
)

const DefaultReadinessTimeout = 150 * time.Second

func main() {
	iopFileNames := []string{os.Args[1]}

	consoleLogger := istioclient.CreateIstioLibraryLogger()

	if err := istioclient.ConfigureIstioLogScopes(); err != nil {
		consoleLogger.LogAndError("Failed to configure Istio log: ", err)
		os.Exit(1)
	}

	printer := istio.NewPrinterForWriter(os.Stdout)

	rc, err := kube.DefaultRestConfig("", "", func(config *rest.Config) {
		config.QPS = 50
		config.Burst = 100
	})
	if err != nil {
		consoleLogger.LogAndError("Failed to create default rest config: ", err)
		os.Exit(1)
	}

	cliClient, err := kube.NewCLIClient(kube.NewClientConfigForRestConfig(rc))
	if err != nil {
		consoleLogger.LogAndError("Failed to create Istio CLI client: ", err)
		os.Exit(1)
	}

	err = k8sversion.IsK8VersionSupported(cliClient, consoleLogger)
	if err != nil {
		consoleLogger.LogAndError("Check failed for minimum supported Kubernetes version: ", err)
		os.Exit(1)
	}

	// We don't want to verify after installation, because it is unreliable
	installArgs := &istio.InstallArgs{ReadinessTimeout: DefaultReadinessTimeout, SkipConfirmation: true, Verify: false, InFilenames: iopFileNames}

	err = istio.Install(cliClient, &istio.RootArgs{}, installArgs, os.Stdout, consoleLogger, printer)
	if err != nil {
		consoleLogger.LogAndError("Istio install error: ", err)
		os.Exit(1)
	}
}
