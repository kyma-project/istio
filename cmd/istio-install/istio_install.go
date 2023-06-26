package main

import (
	istio "istio.io/istio/operator/cmd/mesh"
	"istio.io/istio/operator/pkg/util/clog"
	istiolog "istio.io/pkg/log"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Istio installation external process entry point
// we decided to run istio install in external process because of goroutines leak in 1.18
// it should be fixed in 1.19 so check if  to revert this change after release
// BECAREFUL the struct and functions bellow are a copy of original ones in kyma/istio
// copied to limit the changes provided to main code because of planned reverse

type IstioClient struct {
	istioLogOptions *istiolog.Options
	consoleLogger   *clog.ConsoleLogger
	printer         istio.Printer
}

func newIstioClient() *IstioClient {
	istioLogOptions := initializeLog()
	registeredScope := istiolog.RegisterScope("installation", "installation", 0)
	consoleLogger := clog.NewConsoleLogger(os.Stdout, os.Stderr, registeredScope)
	printer := istio.NewPrinterForWriter(os.Stdout)

	return &IstioClient{istioLogOptions: istioLogOptions, consoleLogger: consoleLogger, printer: printer}
}

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
	c := newIstioClient()

	// We don't want to verify after installation, because it is unreliable
	installArgs := &istio.InstallArgs{SkipConfirmation: true, Verify: false, InFilenames: iopFileNames}

	if err := istio.Install(&istio.RootArgs{}, installArgs, c.istioLogOptions, os.Stdout, c.consoleLogger, c.printer); err != nil {
		ctrl.Log.Error(err, "Error occured during istio installation in external process")
		os.Exit(1)
	}
}
