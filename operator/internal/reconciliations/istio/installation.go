package istio

import (
	"context"
	"encoding/json"

	ctrl "sigs.k8s.io/controller-runtime"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	"github.com/masterminds/semver"
	appsv1 "k8s.io/api/apps/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Installation struct {
	Client         IstioClient
	IstioVersion   string
	IstioImageBase string
}

const (
	LastAppliedConfiguration string = "operator.kyma-project.io/lastAppliedConfiguration"
)

// Reconcile setup configuration and runs an Istio installation with merged Istio Operator manifest file.
func (i *Installation) Reconcile(ctx context.Context, istioCR *operatorv1alpha1.Istio, kubeClient client.Client) (ctrl.Result, error) {
	needsInstall, err := configurationChanged(*istioCR)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !needsInstall {
		installedVersions, err := gatherer.ListInstalledIstioRevisions(ctx, kubeClient)
		if err != nil {
			return ctrl.Result{}, err
		}

		if len(installedVersions) > 0 {
			// compare versions with default revision
			needsInstall = !semver.MustParse(i.IstioVersion).Equal(installedVersions["default"])
		} else {
			needsInstall = true
		}
	}

	if !needsInstall {
		ctrl.Log.Info("Install of Istio was skipped")
		return ctrl.Result{}, nil
	}

	mergedIstioOperatorPath, err := merge(istioCR, i.Client.defaultIstioOperatorPath, i.Client.workingDir, TemplateData{IstioVersion: i.IstioVersion, IstioImageBase: i.IstioImageBase})
	if err != nil {
		return ctrl.Result{}, err
	}

	err = i.Client.Install(mergedIstioOperatorPath)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = updateLastAppliedConfiguration(ctx, kubeClient, *istioCR)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func updateLastAppliedConfiguration(ctx context.Context, kubeClient client.Client, cr operatorv1alpha1.Istio) error {
	if cr.Annotations == nil {
		cr.Annotations = make(map[string]string)
	}

	config, err := json.Marshal(cr.Spec)
	if err != nil {
		return err
	}

	cr.Annotations[LastAppliedConfiguration] = string(config)

	return kubeClient.Update(ctx, &cr)
}

func configurationChanged(istioCR operatorv1alpha1.Istio) (bool, error) {
	lastAppliedConfig, ok := istioCR.Annotations[LastAppliedConfiguration]
	if !ok {
		return true, nil
	}

	var lastAppliedIstioCRSpec operatorv1alpha1.IstioSpec
	json.Unmarshal([]byte(lastAppliedConfig), &lastAppliedIstioCRSpec)

	lastAppliedNotNil := lastAppliedIstioCRSpec.Config.NumTrustedProxies != nil
	newNotNil := istioCR.Spec.Config.NumTrustedProxies != nil

	if lastAppliedNotNil != newNotNil {
		return true, nil
	}

	if !lastAppliedNotNil {
		return true, nil
	}

	return *lastAppliedIstioCRSpec.Config.NumTrustedProxies != *istioCR.Spec.Config.NumTrustedProxies, nil
}

func isIstioInstalled(kubeClient client.Client) bool {
	var istiodList appsv1.DeploymentList
	err := kubeClient.List(context.Background(), &istiodList, client.MatchingLabels(gatherer.IstiodAppLabel))
	if err != nil {
		return false
	}

	return len(istiodList.Items) > 0
}
