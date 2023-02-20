package uninstall

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"istio.io/api/operator/v1alpha1"
	"istio.io/istio/istioctl/pkg/tag"
	. "istio.io/istio/operator/cmd/mesh"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"istio.io/istio/operator/pkg/cache"
	"istio.io/istio/operator/pkg/helmreconciler"
	"istio.io/istio/operator/pkg/object"
	"istio.io/istio/operator/pkg/translate"
	"istio.io/istio/operator/pkg/util/clog"
	"istio.io/istio/operator/pkg/util/progress"
	proxyinfo "istio.io/istio/pkg/proxy"
	"istio.io/pkg/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type UninstallArgs struct {
	// KubeConfigPath is the path to kube config file.
	KubeConfigPath string
	// context is the cluster context in the kube config.
	Context string
	// purge results in deletion of all Istio resources.
	Purge bool
	// revision is the Istio control plane revision the command targets.
	Revision string
	// istioNamespace is the target namespace of istio control plane.
	IstioNamespace string
	// verbose generates verbose output.
	Verbose bool
}

func Uninstall(rootArgs *RootArgs, uiArgs *UninstallArgs, logOpts *log.Options, l clog.Logger) error {
	if err := configLogs(logOpts); err != nil {
		return fmt.Errorf("could not configure logs: %s", err)
	}
	kubeClient, client, err := KubernetesClients(uiArgs.KubeConfigPath, uiArgs.Context, l)
	if err != nil {
		return fmt.Errorf("could not construct k8s clients logs: %s", err)
	}

	if uiArgs.Revision != "" {
		revisions, err := tag.ListRevisionDescriptions(kubeClient)
		if err != nil {
			return fmt.Errorf("could not list revisions: %s", err)
		}
		if _, exists := revisions[uiArgs.Revision]; !exists {
			return errors.New("could not find target revision")
		}
	}

	cache.FlushObjectCaches()
	opts := &helmreconciler.Options{DryRun: rootArgs.DryRun, Log: l, ProgressLog: progress.NewLog()}
	var h *helmreconciler.HelmReconciler

	if uiArgs.Purge && uiArgs.Revision != "" {
		l.LogAndPrint(PurgeWithRevisionOrOperatorSpecifiedWarning)
	}

	emptyiops := &v1alpha1.IstioOperatorSpec{Profile: "empty", Revision: uiArgs.Revision}
	iop, err := translate.IOPStoIOP(emptyiops, "empty", iopv1alpha1.Namespace(emptyiops))
	if err != nil {
		return err
	}
	h, err = helmreconciler.NewHelmReconciler(client, kubeClient, iop, opts)
	if err != nil {
		return fmt.Errorf("failed to create reconciler: %v", err)
	}
	objectsList, err := h.GetPrunedResources(uiArgs.Revision, uiArgs.Purge, "")
	if err != nil {
		return err
	}
	preCheckWarnings(uiArgs, uiArgs.Revision, objectsList, nil, l)

	if err := h.DeleteObjectsList(objectsList, ""); err != nil {
		return fmt.Errorf("failed to delete control plane resources by revision: %v", err)
	}
	opts.ProgressLog.SetState(progress.StateUninstallComplete)
	return nil
}

func preCheckWarnings(uiArgs *UninstallArgs, rev string, resourcesList []*unstructured.UnstructuredList, objectsList object.K8sObjects, l clog.Logger) {
	pids, err := proxyinfo.GetIDsFromProxyInfo(uiArgs.KubeConfigPath, uiArgs.Context, rev, uiArgs.IstioNamespace)
	if err != nil {
		l.LogAndError(err.Error())
	}
	message := ""
	if uiArgs.Purge {
		message += AllResourcesRemovedWarning
	} else {
		rmListString, gwList := constructResourceListOutput(resourcesList, objectsList)
		if rmListString == "" {
			l.LogAndPrint(NoResourcesRemovedWarning)
			return
		}
		if uiArgs.Verbose {
			message += fmt.Sprintf("The following resources will be pruned from the cluster: %s\n",
				rmListString)
		}

		if len(pids) != 0 && rev != "" {
			message += fmt.Sprintf("There are still %d proxies pointing to the control plane revision %s\n", len(pids), rev)
			// just print the count only if there is a large list of proxies
			if len(pids) <= 30 {
				message += fmt.Sprintf("%s\n", strings.Join(pids, "\n"))
			}
			message += "If you proceed with the uninstall, these proxies will become detached from any control plane" +
				" and will not function correctly.\n"
		}
		if gwList != "" {
			message += fmt.Sprintf(GatewaysRemovedWarning, gwList)
		}
	}
	l.LogAndPrint(message)
	return
}

func constructResourceListOutput(resourcesList []*unstructured.UnstructuredList, objectsList object.K8sObjects) (string, string) {
	var items []unstructured.Unstructured
	if objectsList != nil {
		items = objectsList.UnstructuredItems()
	}
	for _, usList := range resourcesList {
		items = append(items, usList.Items...)
	}
	kindNameMap := make(map[string][]string)
	for _, o := range items {
		nameList := kindNameMap[o.GetKind()]
		if nameList == nil {
			kindNameMap[o.GetKind()] = []string{}
		}
		kindNameMap[o.GetKind()] = append(kindNameMap[o.GetKind()], o.GetName())
	}
	if len(kindNameMap) == 0 {
		return "", ""
	}
	output, gwlist := "", []string{}
	for kind, name := range kindNameMap {
		output += fmt.Sprintf("%s: %s. ", kind, strings.Join(name, ", "))
		if kind == "Deployment" {
			for _, n := range name {
				if strings.Contains(n, "gateway") {
					gwlist = append(gwlist, n)
				}
			}
		}
	}
	return output, strings.Join(gwlist, ", ")
}

var logMutex = sync.Mutex{}

func configLogs(opt *log.Options) error {
	logMutex.Lock()
	defer logMutex.Unlock()
	op := []string{"stderr"}
	opt2 := *opt
	opt2.OutputPaths = op
	opt2.ErrorOutputPaths = op

	return log.Configure(&opt2)
}
