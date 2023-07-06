package istio

import (
	"context"
	"fmt"

	ingressgateway "github.com/kyma-project/istio/operator/internal/ingress-gateway"
	"github.com/kyma-project/istio/operator/internal/resources"
	sidecarRemover "github.com/kyma-project/istio/operator/pkg/lib/sidecars/remove"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"

	"github.com/kyma-project/istio/operator/internal/manifest"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Installation struct {
	IstioClient    LibraryClient
	IstioVersion   string
	IstioImageBase string
	Client         client.Client
	Merger         manifest.Merger
}

const (
	LastAppliedConfiguration string = "operator.kyma-project.io/lastAppliedConfiguration"
	installationFinalizer    string = "istios.operator.kyma-project.io/istio-installation"
)

// Reconcile runs Istio reconciliation to install, upgrade or uninstall Istio and returns the updated Istio CR.
func (i *Installation) Reconcile(ctx context.Context, istioCR operatorv1alpha1.Istio, istioResourceListPath string) (operatorv1alpha1.Istio, error) {
	istioTag := fmt.Sprintf("%s-%s", i.IstioVersion, i.IstioImageBase)

	shouldInstallIstio, err := shouldInstall(istioCR, istioTag)
	if err != nil {
		ctrl.Log.Error(err, "Error evaluating Istio CR changes")
		return istioCR, err
	}

	if shouldInstallIstio {
		if !hasInstallationFinalizer(istioCR) {
			controllerutil.AddFinalizer(&istioCR, installationFinalizer)
			if err := i.Client.Update(ctx, &istioCR); err != nil {
				return istioCR, err
			}
		}

		ctrl.Log.Info("Starting istio install", "istio version", i.IstioVersion, "istio image", i.IstioImageBase)

		// To have a better visibility of the manager state during install and upgrade, we update the status to Processing
		_, err = status.Update(ctx, i.Client, &istioCR, operatorv1alpha1.Processing, metav1.Condition{})
		if err != nil {
			return istioCR, err
		}

		clusterConfiguration, err := clusterconfig.EvaluateClusterConfiguration(ctx, i.Client)
		if err != nil {
			return istioCR, err
		}

		// As we define default IstioOperator values in a templated manifest, we need to apply the istio version and values from
		// Istio CR to this default configuration to get the final IstoOperator that is used for installing and updating Istio.
		templateData := manifest.TemplateData{IstioVersion: i.IstioVersion, IstioImageBase: i.IstioImageBase}

		cSize, err := clusterconfig.EvaluateClusterSize(context.Background(), i.Client)
		if err != nil {
			ctrl.Log.Error(err, "Error occurred during evaluation of cluster size")
			return istioCR, err
		}

		ctrl.Log.Info("Installing istio with", "profile", cSize.String())

		mergedIstioOperatorPath, err := i.Merger.Merge(cSize.DefaultManifestPath(), &istioCR, templateData, clusterConfiguration)
		if err != nil {
			return istioCR, err
		}

		ingressGatewayNeedsRestart, err := ingressgateway.NeedsRestart(ctx, i.Client, &istioCR)
		if err != nil {
			return istioCR, err
		}

		err = i.IstioClient.Install(mergedIstioOperatorPath)
		if err != nil {
			return istioCR, err
		}

		err = addWardenValidationAndDisclaimer(ctx, i.Client)
		if err != nil {
			return istioCR, err
		}

		version, err := gatherer.GetIstioPodsVersion(ctx, i.Client)
		if err != nil {
			return istioCR, err
		}

		if i.IstioVersion != version {
			return istioCR, fmt.Errorf("istio-system pods version: %s do not match target version: %s", version, i.IstioVersion)
		}

		ctrl.Log.Info("Istio install completed")

		if ingressGatewayNeedsRestart {
			ctrl.Log.Info("Restarting istio-ingressgateway")
			err = ingressgateway.RestartDeployment(ctx, i.Client)
			if err != nil {
				return istioCR, err
			}
		}

		// We use the installation finalizer to track if the deletion was already executed so can make the uninstallation process more reliable.
	} else if shouldDelete(istioCR) && hasInstallationFinalizer(istioCR) {
		ctrl.Log.Info("Starting istio uninstall")

		_, err = status.Update(ctx, i.Client, &istioCR, operatorv1alpha1.Deleting, metav1.Condition{})
		if err != nil {
			return istioCR, err
		}

		istioResourceFinder, err := resources.NewIstioResourcesFinderFromConfigYaml(ctx, i.Client, ctrl.Log, istioResourceListPath)
		if err != nil {
			return istioCR, err
		}

		clientResources, err := istioResourceFinder.FindUserCreatedIstioResources()
		if err != nil {
			return istioCR, err
		}
		if len(clientResources) > 0 {
			return istioCR, fmt.Errorf("could not delete Istio module instance since there are %d customer created resources present", len(clientResources))
		}

		err = i.IstioClient.Uninstall(ctx)
		if err != nil {
			return istioCR, err
		}

		warnings, err := sidecarRemover.RemoveSidecars(ctx, i.Client, &ctrl.Log)
		if err != nil {
			return istioCR, err
		}

		if len(warnings) > 0 {
			for _, w := range warnings {
				ctrl.Log.Info("Removing sidecar warning:", "name", w.Name, "namespace", w.Namespace, "kind", w.Kind, "message", w.Message)
			}
		}

		controllerutil.RemoveFinalizer(&istioCR, installationFinalizer)
		if err := i.Client.Update(ctx, &istioCR); err != nil {
			ctrl.Log.Error(err, "Error happened during istio installation finalizer removal")
			return istioCR, err
		}

		ctrl.Log.Info("Istio uninstall completed")
	}

	return istioCR, nil
}

func hasInstallationFinalizer(istioCR operatorv1alpha1.Istio) bool {
	return controllerutil.ContainsFinalizer(&istioCR, installationFinalizer)
}
