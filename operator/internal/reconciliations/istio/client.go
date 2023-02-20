package istio

import (
	"os"

	"github.com/kyma-project/istio/operator/pkg/lib/uninstall"
	istio "istio.io/istio/operator/cmd/mesh"
	"istio.io/istio/operator/pkg/util/clog"
	istiolog "istio.io/pkg/log"
)

type IstioClient struct {
	istioLogOptions          *istiolog.Options
	consoleLogger            *clog.ConsoleLogger
	printer                  istio.Printer
	defaultIstioOperatorPath string
	workingDir               string
}

func NewIstioClient(defaultIstioOperatorPath string, workingDir string, istioLogScope string) IstioClient {
	istioLogOptions := initializeLog()
	installerScope := istiolog.RegisterScope(istioLogScope, istioLogScope, 0)
	consoleLogger := clog.NewConsoleLogger(os.Stdout, os.Stderr, installerScope)
	printer := istio.NewPrinterForWriter(os.Stdout)

	return IstioClient{istioLogOptions: istioLogOptions, consoleLogger: consoleLogger, printer: printer, defaultIstioOperatorPath: defaultIstioOperatorPath, workingDir: workingDir}
}

func (c *IstioClient) Install(mergedIstioOperatorPath string) error {
	iopFileNames := make([]string, 0, 1)
	iopFileNames = append(iopFileNames, mergedIstioOperatorPath)
	// We don't want to verify after installation, because it is unreliable
	installArgs := &istio.InstallArgs{SkipConfirmation: true, Verify: false, InFilenames: iopFileNames}

	if err := istio.Install(&istio.RootArgs{}, installArgs, c.istioLogOptions, os.Stdout, c.consoleLogger, c.printer); err != nil {
		return err
	}

	return nil
}

func (c *IstioClient) Uninstall() error {
	uiArgs := &uninstall.UninstallArgs{Purge: true}
	if err := uninstall.Uninstall(&istio.RootArgs{}, uiArgs, c.istioLogOptions, c.consoleLogger); err != nil {
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
	logoptions.SetOutputLevel("uninstaller", istiolog.WarnLevel)
	logoptions.SetOutputLevel("translator", istiolog.WarnLevel)
	logoptions.SetOutputLevel("adsc", istiolog.WarnLevel)
	logoptions.SetOutputLevel("default", istiolog.WarnLevel)
	logoptions.SetOutputLevel("klog", istiolog.WarnLevel)
	logoptions.SetOutputLevel("kube", istiolog.ErrorLevel)

	return logoptions
}
