package istio

import (
	"context"
	"fmt"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"

	"istio.io/api/operator/v1alpha1"
	"istio.io/istio/istioctl/pkg/install/k8sversion"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"istio.io/istio/operator/pkg/cache"
	"istio.io/istio/operator/pkg/helmreconciler"
	"istio.io/istio/operator/pkg/translate"
	"istio.io/istio/operator/pkg/util/progress"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/kube"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	istio "istio.io/istio/operator/cmd/mesh"
	"istio.io/istio/operator/pkg/util/clog"
	istiolog "istio.io/istio/pkg/log"
)

type LibraryClient interface {
	Install(mergedIstioOperatorPath string) error
	Uninstall(ctx context.Context) error
}

type IstioClient struct {
	istioLogOptions *istiolog.Options
	consoleLogger   *clog.ConsoleLogger
	printer         istio.Printer
}

func NewIstioClient() *IstioClient {
	istioLogOptions := initializeLog()
	registeredScope := istiolog.RegisterScope("installation", "installation")
	consoleLogger := clog.NewConsoleLogger(os.Stdout, os.Stderr, registeredScope)
	printer := istio.NewPrinterForWriter(os.Stdout)

	return &IstioClient{istioLogOptions: istioLogOptions, consoleLogger: consoleLogger, printer: printer}
}

func (c *IstioClient) Install(mergedIstioOperatorPath string) error {

	rc, err := kube.DefaultRestConfig("", "", func(config *rest.Config) {
		config.QPS = 50
		config.Burst = 100
	})
	if err != nil {
		return err
	}

	cliClient, err := kube.NewCLIClient(kube.NewClientConfigForRestConfig(rc), "")
	if err != nil {
		return err
	}

	if err := k8sversion.IsK8VersionSupported(cliClient, c.consoleLogger); err != nil {
		return err
	}

	iopFileNames := make([]string, 0, 1)
	iopFileNames = append(iopFileNames, mergedIstioOperatorPath)
	// We don't want to verify after installation, because it is unreliable
	installArgs := &istio.InstallArgs{ReadinessTimeout: 150 * time.Second, SkipConfirmation: true, Verify: false, InFilenames: iopFileNames}

	if err := istio.Install(cliClient, &istio.RootArgs{}, installArgs, c.istioLogOptions, os.Stdout, c.consoleLogger, c.printer); err != nil {
		return err
	}

	return nil
}

func (c *IstioClient) Uninstall(ctx context.Context) error {
	// We don't use any revision capabilities yet
	defaultRevision := ""

	// Since we copied the internal uninstall function, we also need to make sure that Istio's internal logging is correctly configured
	if err := initializeIstioLogSubsystem(c.istioLogOptions); err != nil {
		return fmt.Errorf("could not configure logs: %s", err)
	}

	rc, err := kube.DefaultRestConfig("", "", func(config *rest.Config) {
		config.QPS = 50
		config.Burst = 100
	})
	if err != nil {
		return fmt.Errorf("failed to create default REST config: %v", err)
	}

	kubeClient, err := kube.NewClient(kube.NewClientConfigForRestConfig(rc), "")
	if err != nil {
		return fmt.Errorf("failed to create Istio kube client: %v", err)
	}

	if err := k8sversion.IsK8VersionSupported(kubeClient, c.consoleLogger); err != nil {
		return fmt.Errorf("check failed for minimum supported Kubernetes version: %v", err)
	}

	ctrlClient, err := client.New(kubeClient.RESTConfig(), client.Options{Scheme: kube.IstioScheme})
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes ctrl client: %v", err)
	}

	cache.FlushObjectCaches()

	emptyiops := &v1alpha1.IstioOperatorSpec{Profile: "empty", Revision: defaultRevision}
	iop, err := translate.IOPStoIOP(emptyiops, "empty", iopv1alpha1.Namespace(emptyiops))
	if err != nil {
		return err
	}

	opts := &helmreconciler.Options{DryRun: false, Log: c.consoleLogger, ProgressLog: progress.NewLog()}
	h, err := helmreconciler.NewHelmReconciler(ctrlClient, kubeClient, iop, opts)
	if err != nil {
		return fmt.Errorf("failed to create reconciler: %v", err)
	}

	objectsList, err := h.GetPrunedResources(defaultRevision, true, "")
	if err != nil {
		return err
	}

	ctrl.Log.Info(istio.AllResourcesRemovedWarning)

	if err := h.DeleteObjectsList(objectsList, ""); err != nil {
		return fmt.Errorf("failed to delete control plane resources by revision: %v", err)
	}
	ctrl.Log.Info("Deletion of istio resources completed")

	deletePolicy := metav1.DeletePropagationForeground
	// We need to manually delete the control plane namespace from Istio because the namespace is not removed by default.
	err = ctrlClient.Delete(ctx, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.IstioSystemNamespace,
		},
	}, &ctrlclient.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil && !apiErrors.IsNotFound(err) {
		return err
	}
	ctrl.Log.Info("Deleted istio control plane namespace", "namespace", constants.IstioSystemNamespace)

	opts.ProgressLog.SetState(progress.StateUninstallComplete)

	return nil
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

var istioLogMutex = sync.Mutex{}

func initializeIstioLogSubsystem(opt *istiolog.Options) error {
	istioLogMutex.Lock()
	defer istioLogMutex.Unlock()
	op := []string{"stderr"}
	opt2 := *opt
	opt2.OutputPaths = op
	opt2.ErrorOutputPaths = op

	return istiolog.Configure(&opt2)
}
