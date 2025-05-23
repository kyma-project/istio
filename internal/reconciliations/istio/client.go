package istio

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"istio.io/istio/istioctl/pkg/install/k8sversion"
	"istio.io/istio/operator/pkg/uninstall"
	"istio.io/istio/operator/pkg/util/progress"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pkg/errors"
	istio "istio.io/istio/operator/cmd/mesh"
	"istio.io/istio/operator/pkg/util/clog"
	istiolog "istio.io/istio/pkg/log"
)

const DefaultClientTimeout = 6 * time.Minute

type libraryClient interface {
	Install(mergedIstioOperatorPath string) error
	Uninstall(ctx context.Context) error
}

type Client struct {
	consoleLogger *clog.ConsoleLogger
	printer       istio.Printer
}

const (
	logScope = "istio-library"
)

func CreateIstioLibraryLogger() *clog.ConsoleLogger {
	registeredScope := istiolog.RegisterScope(logScope, logScope)
	return clog.NewConsoleLogger(os.Stdout, os.Stderr, registeredScope)
}

func NewIstioClient() *Client {
	consoleLogger := CreateIstioLibraryLogger()
	printer := istio.NewPrinterForWriter(os.Stdout)

	return &Client{consoleLogger: consoleLogger, printer: printer}
}

func installIstioInExternalProcess(mergedIstioOperatorPath string) error {
	istioInstallPath, ok := os.LookupEnv("ISTIO_INSTALL_BIN_PATH")
	if !ok {
		istioInstallPath = "./istio_install"
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultClientTimeout)
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

func (c *Client) Install(mergedIstioOperatorPath string) error {
	err := installIstioInExternalProcess(mergedIstioOperatorPath)

	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Uninstall(ctx context.Context) error {
	rc, err := kube.DefaultRestConfig("", "", func(config *rest.Config) {
		config.QPS = 50
		config.Burst = 100
	})
	if err != nil {
		return fmt.Errorf("failed to create default REST config: %w", err)
	}

	kubeClient, err := kube.NewCLIClient(kube.NewClientConfigForRestConfig(rc))
	if err != nil {
		return fmt.Errorf("failed to create Istio kube client: %w", err)
	}

	err = k8sversion.IsK8VersionSupported(kubeClient, c.consoleLogger)
	if err != nil {
		return fmt.Errorf("check failed for minimum supported Kubernetes version: %w", err)
	}

	ctrlClient, err := client.New(kubeClient.RESTConfig(), client.Options{Scheme: kube.IstioScheme})
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes ctrl client: %w", err)
	}

	pl := progress.NewLog()

	objectsList, err := uninstall.GetPrunedResources(kubeClient, "", "", "default", true)
	if err != nil {
		return err
	}

	ctrl.Log.Info(istio.AllResourcesRemovedWarning)

	consoleLogger := CreateIstioLibraryLogger()
	err = ConfigureIstioLogScopes()
	if err != nil {
		consoleLogger.LogAndError("Failed to configure Istio log: ", err)
		return err
	}

	err = uninstall.DeleteObjectsList(kubeClient, false, consoleLogger, objectsList)
	if err != nil {
		return fmt.Errorf("failed to delete control plane resources by revision: %w", err)
	}
	ctrl.Log.Info("Deletion of istio resources completed")

	deletePolicy := metav1.DeletePropagationForeground
	// We need to manually delete the control plane namespace from Istio because the namespace is not removed by default.
	err = ctrlClient.Delete(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.IstioSystemNamespace,
		},
	}, &client.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil && !apiErrors.IsNotFound(err) {
		return err
	}
	ctrl.Log.Info("Deleted istio control plane namespace", "namespace", constants.IstioSystemNamespace)

	pl.SetState(progress.StateUninstallComplete)

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
