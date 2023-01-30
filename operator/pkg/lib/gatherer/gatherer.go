package gatherer

import (
	"context"
	"fmt"

	"github.com/masterminds/semver"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IstioNamespace           string = "istio-system"
	RevisionLabelName        string = "istio.io/rev"
	VersionLabelName         string = "operator.istio.io/version"
	DefaultIstioRevisionName string = "default"
)

var IstiodAppLabel map[string]string = map[string]string{"app": "istiod"}

// GetIstioCR fetches the Istio CR from the cluster using client with supplied name and namespace
func GetIstioCR(ctx context.Context, client client.Client, name string, namespace string) (*v1alpha1.Istio, error) {
	cr := v1alpha1.Istio{}
	err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &cr)
	if err != nil {
		return nil, err
	}

	return &cr, nil
}

// ListIstioCR lists all Istio CRs on the cluster if no namespace is supplied, or from the supplied namespaces
func ListIstioCR(ctx context.Context, kubeclient client.Client, namespace ...string) (*v1alpha1.IstioList, error) {
	list := v1alpha1.IstioList{}

	if len(namespace) == 0 {
		err := kubeclient.List(ctx, &list)
		if err != nil {
			return nil, err
		}
	} else {
		for _, n := range namespace {
			namespacedList := v1alpha1.IstioList{}

			err := kubeclient.List(ctx, &namespacedList, &client.ListOptions{Namespace: n})
			if err != nil {
				return nil, err
			}

			list.Items = append(list.Items, namespacedList.Items...)
		}
	}

	return &list, nil
}

func ListInstalledIstioRevisions(ctx context.Context, kubeclient client.Client) (istioRevisionVersions map[string]*semver.Version, err error) {
	istioRevisionVersions = make(map[string]*semver.Version)

	var istiodList appsv1.DeploymentList
	err = kubeclient.List(ctx, &istiodList, client.MatchingLabels(IstiodAppLabel))
	if err != nil {
		return nil, err
	}

	for _, istiodDeployment := range istiodList.Items {
		version, ok := istiodDeployment.Labels[VersionLabelName]
		if !ok {
			return nil, fmt.Errorf("istiod deployment %s didn't have version label", istiodDeployment.Name)
		}

		revision, ok := istiodDeployment.Labels[RevisionLabelName]
		if !ok {
			return nil, fmt.Errorf("istiod deployment %s didn't have revision label", istiodDeployment.Name)
		}

		// Istio version label sometimes is set to unknown for unknown reason.
		// TODO: Make the logic more resilient by e.g. using Istiod image tag
		if version != "unknown" {
			semverVersion, err := semver.NewVersion(version)
			if err != nil {
				return nil, err
			}

			istioRevisionVersions[revision] = semverVersion
		}
	}

	return istioRevisionVersions, nil
}
