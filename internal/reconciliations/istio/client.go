package istio

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	istio "istio.io/istio/operator/cmd/mesh"
	"istio.io/istio/operator/pkg/util/clog"
	istiolog "istio.io/istio/pkg/log"
)

type libraryClient interface {
	Install(mergedIstioOperatorPath string) error
	Uninstall(ctx context.Context) error
}

type IstioClient struct {
	consoleLogger *clog.ConsoleLogger
	printer       istio.Printer
}

const logScope = "istio-library"

func CreateIstioLibraryLogger() *clog.ConsoleLogger {
	registeredScope := istiolog.RegisterScope(logScope, logScope)
	return clog.NewConsoleLogger(os.Stdout, os.Stderr, registeredScope)
}

func NewIstioClient() *IstioClient {
	consoleLogger := CreateIstioLibraryLogger()
	printer := istio.NewPrinterForWriter(os.Stdout)

	return &IstioClient{consoleLogger: consoleLogger, printer: printer}
}

func installIstioInExternalProcess(mergedIstioOperatorPath string) error {
	istioInstallPath, ok := os.LookupEnv("ISTIO_INSTALL_BIN_PATH")
	if !ok {
		istioInstallPath = "./istio_install"
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*6)
	defer cancel()

	cmd := exec.CommandContext(ctx, istioInstallPath, mergedIstioOperatorPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		// We should not return the error of the external process, because it is always "exit status 1" and we do
		// not want to show such an error in the resource status
		return errors.New("Istio installation resulted in an error")
	}

	return nil
}

func (c *IstioClient) Install(mergedIstioOperatorPath string) error {
	err := installIstioInExternalProcess(mergedIstioOperatorPath)

	if err != nil {
		return err
	}

	return nil
}

func (c *IstioClient) Uninstall(ctx context.Context) error {
	// TODO: Implement uninstallation
	//// We don't use any revision capabilities yet
	//defaultRevision := ""
	//
	//rc, err := kube.DefaultRestConfig("", "", func(config *rest.Config) {
	//	config.QPS = 50
	//	config.Burst = 100
	//})
	//if err != nil {
	//	return fmt.Errorf("failed to create default REST config: %v", err)
	//}
	//
	//kubeClient, err := kube.NewClient(kube.NewClientConfigForRestConfig(rc), "")
	//if err != nil {
	//	return fmt.Errorf("failed to create Istio kube client: %v", err)
	//}
	//
	//if err := k8sversion.IsK8VersionSupported(kubeClient, c.consoleLogger); err != nil {
	//	return fmt.Errorf("check failed for minimum supported Kubernetes version: %v", err)
	//}
	//
	//ctrlClient, err := client.New(kubeClient.RESTConfig(), client.Options{Scheme: kube.IstioScheme})
	//if err != nil {
	//	return fmt.Errorf("failed to create Kubernetes ctrl client: %v", err)
	//}
	//
	//emptyiops := &iopv1alpha1.IstioOperatorSpec{Profile: "empty", Revision: defaultRevision}
	//iop, err := translate.IOPStoIOP(emptyiops, "empty", iopv1alpha1.Namespace(emptyiops))
	//if err != nil {
	//	return err
	//}
	//
	//opts := &helmreconciler.Options{DryRun: false, Log: c.consoleLogger, ProgressLog: progress.NewLog()}
	//h, err := helmreconciler.NewHelmReconciler(ctrlClient, kubeClient, iop, opts)
	//if err != nil {
	//	return fmt.Errorf("failed to create reconciler: %v", err)
	//}
	//
	//objectsList, err := h.GetPrunedResources(defaultRevision, true, "")
	//if err != nil {
	//	return err
	//}
	//
	//ctrl.Log.Info(istio.AllResourcesRemovedWarning)
	//
	//if err := h.DeleteObjectsList(objectsList, ""); err != nil {
	//	return fmt.Errorf("failed to delete control plane resources by revision: %v", err)
	//}
	//ctrl.Log.Info("Deletion of istio resources completed")
	//
	//deletePolicy := metav1.DeletePropagationForeground
	//// We need to manually delete the control plane namespace from Istio because the namespace is not removed by default.
	//err = ctrlClient.Delete(ctx, &v1.Namespace{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name: constants.IstioSystemNamespace,
	//	},
	//}, &ctrlclient.DeleteOptions{
	//	PropagationPolicy: &deletePolicy,
	//})
	//if err != nil && !apiErrors.IsNotFound(err) {
	//	return err
	//}
	//ctrl.Log.Info("Deleted istio control plane namespace", "namespace", constants.IstioSystemNamespace)
	//
	//opts.ProgressLog.SetState(progress.StateUninstallComplete)

	return nil
}

func ConfigureIstioLogScopes() error {
	o := istiolog.DefaultOptions()
	o.SetDefaultOutputLevel(logScope, istiolog.WarnLevel)
	o.SetDefaultOutputLevel("analysis", istiolog.WarnLevel)
	o.SetDefaultOutputLevel("translator", istiolog.WarnLevel)
	o.SetDefaultOutputLevel("adsc", istiolog.WarnLevel)
	o.SetDefaultOutputLevel("klog", istiolog.WarnLevel)
	// These scopes are too noisy even at warning level
	o.SetDefaultOutputLevel("validation", istiolog.ErrorLevel)
	o.SetDefaultOutputLevel("processing", istiolog.ErrorLevel)
	o.SetDefaultOutputLevel("kube", istiolog.ErrorLevel)

	return istiolog.Configure(o)
}
