// Istio install was identified to be prone to memory leaks.
// As so, we want to separate the dependency istio installation by running it in a separate process.
// This is the file responsible for executing call to istio install
package main

import (
	"os"
	"time"

	"istio.io/istio/istioctl/pkg/install/k8sversion"
	istio "istio.io/istio/operator/cmd/mesh"
	"istio.io/istio/operator/pkg/util/clog"
	"istio.io/istio/pkg/kube"
	istiolog "istio.io/istio/pkg/log"
	"k8s.io/client-go/rest"
)

func initializeLog() *istiolog.Options {
	logoptions := istiolog.DefaultOptions()
	logoptions.SetOutputLevel("validation", istiolog.ErrorLevel)
	logoptions.SetOutputLevel("processing", istiolog.ErrorLevel)
	logoptions.SetOutputLevel("analysis", istiolog.WarnLevel)
	logoptions.SetOutputLevel("installation", istiolog.WarnLevel)
	logoptions.SetOutputLevel("translator", istiolog.WarnLevel)
	logoptions.SetOutputLevel("adsc", istiolog.WarnLevel)
	logoptions.SetOutputLevel("default", istiolog.WarnLevel)
	logoptions.SetOutputLevel("klog", istiolog.WarnLevel)
	logoptions.SetOutputLevel("kube", istiolog.ErrorLevel)

	return logoptions
}

func main() {
	iopFileNames := []string{os.Args[1]}
	compatibilityFlag := os.Args[2]

	var setOptions []string

	if compatibilityFlag != "" {
		setOptions = append(setOptions, compatibilityFlag)
	}

	istioLogOptions := initializeLog()
	registeredScope := istiolog.RegisterScope("installation", "installation")
	consoleLogger := clog.NewConsoleLogger(os.Stdout, os.Stderr, registeredScope)
	printer := istio.NewPrinterForWriter(os.Stdout)

	rc, err := kube.DefaultRestConfig("", "", func(config *rest.Config) {
		config.QPS = 50
		config.Burst = 100
	})
	if err != nil {
		consoleLogger.LogAndError("Failed to create default rest config: ", err)
		os.Exit(1)
	}

	cliClient, err := kube.NewCLIClient(kube.NewClientConfigForRestConfig(rc), "")
	if err != nil {
		consoleLogger.LogAndError("Failed to create Istio CLI client: ", err)
		os.Exit(1)
	}

	if err := k8sversion.IsK8VersionSupported(cliClient, consoleLogger); err != nil {
		consoleLogger.LogAndError("Check failed for minimum supported Kubernetes version: ", err)
		os.Exit(1)
	}

	// We don't want to verify after installation, because it is unreliable
	installArgs := &istio.InstallArgs{ReadinessTimeout: 150 * time.Second, SkipConfirmation: true, Verify: false, InFilenames: iopFileNames, Set: setOptions}

	if err := istio.Install(cliClient, &istio.RootArgs{}, installArgs, istioLogOptions, os.Stdout, consoleLogger, printer); err != nil {
		consoleLogger.LogAndError("Istio install error: ", err)
		os.Exit(1)
	}
}
