// Istio install was identified to be prone to memory leaks.
// As so, we want to separate the dependency istio installation by running it in a separate process.
// This is the file responsible for executing call to istio install
package main

import (
	"fmt"
	"istio.io/istio/operator/pkg/install"
	"istio.io/istio/operator/pkg/render"
	"istio.io/istio/operator/pkg/util/clog"
	"istio.io/istio/operator/pkg/util/progress"
	"os"
	"time"

	istioclient "github.com/kyma-project/istio/operator/internal/reconciliations/istio"

	"istio.io/istio/istioctl/pkg/install/k8sversion"
	"istio.io/istio/pkg/kube"
	"k8s.io/client-go/rest"
)

const (
	DefaultReadinessTimeout = 150 * time.Second
)

func main() {
	consoleLogger := istioclient.CreateIstioLibraryLogger()

	if len(os.Args) <= 1 {
		consoleLogger.LogAndError("IOP file names must be provided as first parameter")
		os.Exit(1)
	}
	iopFileNames := []string{os.Args[1]}

	if err := istioclient.ConfigureIstioLogScopes(); err != nil {
		consoleLogger.LogAndError("Failed to configure Istio log: ", err)
		os.Exit(1)
	}

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

	if err = k8sversion.IsK8VersionSupported(cliClient, consoleLogger); err != nil {
		consoleLogger.LogAndError("Check failed for minimum supported Kubernetes version: ", err)
		os.Exit(1)
	}

	if err = IstioInstall(cliClient, iopFileNames, consoleLogger); err != nil {
		consoleLogger.LogAndError("Istio install error: ", err)
		os.Exit(1)
	}
}

func IstioInstall(kubeClient kube.CLIClient, iopFileNames []string, l clog.Logger) error {
	manifests, vals, err := render.GenerateManifest(iopFileNames, []string{}, false, kubeClient, l)
	if err != nil {
		return fmt.Errorf("generate config: %v", err)
	}

	i := install.Installer{
		Force:          false,
		DryRun:         false,
		SkipWait:       false,
		Kube:           kubeClient,
		WaitTimeout:    DefaultReadinessTimeout,
		Logger:         l,
		Values:         vals,
		ProgressLogger: progress.NewLog(),
	}
	if err := i.InstallManifests(manifests); err != nil {
		return fmt.Errorf("failed to install manifests: %v", err)
	}

	return nil
}
