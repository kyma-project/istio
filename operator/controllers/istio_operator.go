package controllers

import (
	"os"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"

	istio "istio.io/istio/operator/cmd/mesh"
	"istio.io/istio/operator/pkg/util/clog"
	istiolog "istio.io/pkg/log"
)

func initializeLog() *istiolog.Options {
	logoptions := istiolog.DefaultOptions()
	logoptions.SetOutputLevel("validation", istiolog.ErrorLevel)
	logoptions.SetOutputLevel("processing", istiolog.ErrorLevel)
	logoptions.SetOutputLevel("analysis", istiolog.WarnLevel)
	logoptions.SetOutputLevel("installer", istiolog.WarnLevel)
	logoptions.SetOutputLevel("translator", istiolog.WarnLevel)
	logoptions.SetOutputLevel("adsc", istiolog.WarnLevel)
	logoptions.SetOutputLevel("default", istiolog.WarnLevel)
	logoptions.SetOutputLevel("klog", istiolog.WarnLevel)
	logoptions.SetOutputLevel("kube", istiolog.ErrorLevel)
	return logoptions
}

func ensureIstioOperator(istioSpec operatorv1alpha1.Istio) error {
	istioLogOptions := initializeLog()
	installerScope := istiolog.RegisterScope("installer", "installer", 0)
	consoleLogger := clog.NewConsoleLogger(os.Stdout, os.Stderr, installerScope)
	printer := istio.NewPrinterForWriter(os.Stdout)
	iopFileName := "./hack/default-istio-operator.yaml"
	iopFileNames := make([]string, 0, 1)
	iopFileNames = append(iopFileNames, iopFileName)
	installArgs := &istio.InstallArgs{SkipConfirmation: true, Verify: true, InFilenames: iopFileNames}
	if err := istio.Install(&istio.RootArgs{}, installArgs, istioLogOptions, os.Stdout, consoleLogger, printer); err != nil {
		return err
	}

	return nil
}
