package istio

import (
	"context"
	"encoding/json"

	ctrl "sigs.k8s.io/controller-runtime"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	"github.com/masterminds/semver"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Installation struct {
	Client         IstioClient
	IstioVersion   string
	IstioImageBase string
}

const (
	ConfigMapName                 string = "kyma-istio-status"
	ConfigMapNamespace            string = "kyma-system"
	LastAppliedConfigurationField string = "lastAppliedConfiguration"
)

// Reconcile setup configuration and runs an Istio installation with merged Istio Operator manifest file.
func (i *Installation) Reconcile(ctx context.Context, istioCR *operatorv1alpha1.Istio, kubeClient client.Client) (ctrl.Result, error) {
	lastAppliedCM, err := getInstalationStatusCM(ctx, kubeClient)
	if err != nil {
		return ctrl.Result{}, err
	}

	needsInstall, err := configurationChanged(lastAppliedCM, *istioCR)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !needsInstall {
		installedVersions, err := gatherer.ListInstalledIstioRevisions(ctx, kubeClient)
		if err != nil {
			return ctrl.Result{}, err
		}

		if len(installedVersions) > 0 {
			if semver.MustParse(i.IstioVersion).LessThan(installedVersions["default"]) {
				return ctrl.Result{}, nil
			}
			if len(installedVersions) > 0 {
				// compare versions and make a default revision
				needsInstall = !semver.MustParse(i.IstioVersion).Equal(installedVersions["default"])
			} else {
				needsInstall = true
			}
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

	lastAppliedIstioCR, err := json.Marshal(istioCR.Spec)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = updateInstallationStatusConfigMap(ctx, kubeClient, lastAppliedCM, lastAppliedIstioCR)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func updateInstallationStatusConfigMap(ctx context.Context, kubeClient client.Client, cm *corev1.ConfigMap, newConfiguration []byte) error {
	if cm == nil {
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ConfigMapName,
				Namespace: ConfigMapNamespace,
			},
			BinaryData: map[string][]byte{
				LastAppliedConfigurationField: newConfiguration,
			},
		}

		return kubeClient.Create(ctx, cm)
	}
	cm.BinaryData = map[string][]byte{
		LastAppliedConfigurationField: newConfiguration,
	}

	return kubeClient.Update(ctx, cm)
}

func getInstalationStatusCM(ctx context.Context, kubeClient client.Client) (*corev1.ConfigMap, error) {
	var lastAppliedConfigurationCM corev1.ConfigMap
	err := kubeClient.Get(ctx, types.NamespacedName{Namespace: ConfigMapNamespace, Name: ConfigMapName}, &lastAppliedConfigurationCM)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &lastAppliedConfigurationCM, nil
}

func configurationChanged(lastAppliedConfigurationCM *corev1.ConfigMap, istioCR operatorv1alpha1.Istio) (bool, error) {
	if lastAppliedConfigurationCM == nil {
		return true, nil
	}
	var lastAppliedIstioCRSpec operatorv1alpha1.IstioSpec
	json.Unmarshal(lastAppliedConfigurationCM.BinaryData[LastAppliedConfigurationField], &lastAppliedIstioCRSpec)

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
