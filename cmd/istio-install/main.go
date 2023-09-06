// Due to memory leak in istio 1.18 we temporarily move Istio install call to external process
// This is the file responsible for executing call to istio install
package main

import (
	istio "istio.io/istio/operator/cmd/mesh"
	"istio.io/istio/operator/pkg/util/clog"
	istiolog "istio.io/istio/pkg/log"
	"os"
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

	istioLogOptions := initializeLog()
	registeredScope := istiolog.RegisterScope("installation", "installation")
	consoleLogger := clog.NewConsoleLogger(os.Stdout, os.Stderr, registeredScope)
	printer := istio.NewPrinterForWriter(os.Stdout)

	// We don't want to verify after installation, because it is unreliable
	installArgs := &istio.InstallArgs{SkipConfirmation: true, Verify: false, InFilenames: iopFileNames}

	if err := istio.Install(&istio.RootArgs{}, installArgs, istioLogOptions, os.Stdout, consoleLogger, printer); err != nil {
		consoleLogger.LogAndError("Istio install error: ", err)
		os.Exit(1)
	}
}
