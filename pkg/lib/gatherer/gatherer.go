package gatherer

import (
	"context"
	"errors"
	"fmt"

	"github.com/distribution/reference"

	"github.com/masterminds/semver"
	"golang.org/x/exp/slices"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RevisionLabelName string = "istio.io/rev"
	VersionLabelName  string = "operator.istio.io/version"
	IstioNamespace    string = "istio-system"
)

var IstiodAppLabel = map[string]string{"app": "istiod"}
var NoVersion semver.Version

// GetIstioCR fetches the Istio CR from the cluster using client with supplied name and namespace
func GetIstioCR(ctx context.Context, client client.Client, name string, namespace string) (*v1alpha2.Istio, error) {
	cr := v1alpha2.Istio{}
	err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &cr)
	if err != nil {
		return nil, err
	}

	return &cr, nil
}

// ListIstioCR lists all Istio CRs on the cluster if no namespace is supplied, or from the supplied namespaces
func ListIstioCR(ctx context.Context, kubeClient client.Client, namespace ...string) (*v1alpha2.IstioList, error) {
	list := v1alpha2.IstioList{}

	if len(namespace) == 0 {
		err := kubeClient.List(ctx, &list)
		if err != nil {
			return nil, err
		}
	} else {
		for _, n := range namespace {
			namespacedList := v1alpha2.IstioList{}

			err := kubeClient.List(ctx, &namespacedList, &client.ListOptions{Namespace: n})
			if err != nil {
				return nil, err
			}

			list.Items = append(list.Items, namespacedList.Items...)
		}
	}

	return &list, nil
}

// ListIstioCPPods lists all Istio control plane pods
func ListIstioCPPods(ctx context.Context, kubeClient client.Client) (podsList *v1.PodList, err error) {
	list := v1.PodList{}
	err = kubeClient.List(ctx, &list, &client.ListOptions{Namespace: IstioNamespace})
	if err != nil {
		return nil, err
	}

	return &list, err
}

func ListInstalledIstioRevisions(ctx context.Context, kubeClient client.Client) (istioRevisionVersions map[string]*semver.Version, err error) {
	istioRevisionVersions = make(map[string]*semver.Version)

	var istiodList appsv1.DeploymentList
	err = kubeClient.List(ctx, &istiodList, client.MatchingLabels(IstiodAppLabel))
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

func GetIstioPodsVersion(ctx context.Context, kubeClient client.Client) (string, error) {
	pods, err := ListIstioCPPods(ctx, kubeClient)
	if err != nil {
		return "", err
	}
	currentVersion := &NoVersion
	containersToCheck := []string{"discovery", "istio-proxy", "install-cni"}
	for _, pod := range pods.Items {
		if pod.ObjectMeta.DeletionTimestamp != nil {
			continue
		}
		for _, container := range pod.Spec.Containers {
			if !slices.Contains(containersToCheck, container.Name) {
				continue
			}
			version, err := getImageVersion(container.Image)
			if err != nil {
				return "", err
			}
			if currentVersion.Compare(&NoVersion) == 0 {
				currentVersion = version
				continue
			} else if currentVersion.Compare(version) != 0 {
				return "", fmt.Errorf("Image version of Pod %s %s do not match other Pods version %s", pod.Name, version.String(), currentVersion.String())
			}
		}
	}
	if currentVersion.Compare(&NoVersion) == 0 {
		return "", errors.New("Unable to obtain installed Istio image version")
	}
	return currentVersion.String(), nil
}

func VerifyIstioPodsVersion(ctx context.Context, kubeClient client.Client, istioOperatorVersion string) error {
	podsVersion, err := GetIstioPodsVersion(ctx, kubeClient)
	if err != nil {
		return err
	}
	if podsVersion != istioOperatorVersion {
		return fmt.Errorf("istio-system Pods version %s do not match istio operator version %s", podsVersion, istioOperatorVersion)
	}
	return nil
}

func getImageVersion(image string) (*semver.Version, error) {
	matches := reference.ReferenceRegexp.FindStringSubmatch(image)
	if len(matches) < 3 {
		return &NoVersion, fmt.Errorf("Unable to parse container image reference: %s", image)
	}
	version, err := semver.NewVersion(matches[2])
	if err != nil {
		return &NoVersion, err
	}
	noPreleaseVersion, err := version.SetPrerelease("")
	if err != nil {
		return &NoVersion, err
	}
	return &noPreleaseVersion, nil
}
