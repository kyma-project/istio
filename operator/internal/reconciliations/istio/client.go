package istio

import (
	"os"

	istio "istio.io/istio/operator/cmd/mesh"
	"istio.io/istio/operator/pkg/util/clog"
	istiolog "istio.io/pkg/log"
)

//go:generate mockery --name=IstioClient --output=mocks --outpkg=mocks --case=underscore
type IstioClient struct {
	istioLogOptions *istiolog.Options
	consoleLogger   *clog.ConsoleLogger
	printer         istio.Printer
}

func NewIstioClient() IstioClient {
	istioLogOptions := initializeLog()
	installerScope := istiolog.RegisterScope("installer", "installer", 0)
	consoleLogger := clog.NewConsoleLogger(os.Stdout, os.Stderr, installerScope)
	printer := istio.NewPrinterForWriter(os.Stdout)

	return IstioClient{istioLogOptions: istioLogOptions, consoleLogger: consoleLogger, printer: printer}
}

func (c *IstioClient) Install(istioOpertator string) error {
	iopFileNames := make([]string, 0, 1)
	iopFileNames = append(iopFileNames, istioOpertator)
	installArgs := &istio.InstallArgs{SkipConfirmation: true, Verify: true, InFilenames: iopFileNames}
	if err := istio.Install(&istio.RootArgs{}, installArgs, c.istioLogOptions, os.Stdout, c.consoleLogger, c.printer); err != nil {
		return err
	}

	return nil
}

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
