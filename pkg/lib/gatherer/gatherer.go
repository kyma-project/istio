package gatherer

import (
	"context"
	"errors"
	"fmt"

	"github.com/distribution/reference"
	"k8s.io/apimachinery/pkg/labels"

	"slices"

	"github.com/masterminds/semver"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
)

const (
	RevisionLabelName string = "istio.io/rev"
	VersionLabelName  string = "operator.istio.io/version"
	IstioNamespace    string = "istio-system"
)

// GetIstioCR fetches the Istio CR from the cluster using client with supplied name and namespace.
func GetIstioCR(ctx context.Context, client client.Client, name string, namespace string) (*v1alpha2.Istio, error) {
	cr := v1alpha2.Istio{}
	err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &cr)
	if err != nil {
		return nil, err
	}

	return &cr, nil
}

// ListIstioCR lists all Istio CRs on the cluster if no namespace is supplied, or from the supplied namespaces.
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

// ListIstioCPPods lists all Istio control plane pods.
func ListIstioCPPods(ctx context.Context, kubeClient client.Client) (*v1.PodList, error) {
	list := v1.PodList{}
	err := kubeClient.List(ctx, &list, &client.ListOptions{
		Namespace: IstioNamespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"kyma-project.io/module": "istio",
		}),
	})
	if err != nil {
		return nil, err
	}

	return &list, err
}

func ListInstalledIstioRevisions(ctx context.Context, kubeClient client.Client) (map[string]*semver.Version, error) {
	istioRevisionVersions := make(map[string]*semver.Version)

	var istiodList appsv1.DeploymentList
	istiodLabels := map[string]string{"app": "istiod"}
	err := kubeClient.List(ctx, &istiodList, client.MatchingLabels(istiodLabels))
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
			semverVersion, versionErr := semver.NewVersion(version)
			if versionErr != nil {
				return nil, versionErr
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
	currentVersion := &semver.Version{}
	containersToCheck := []string{"discovery", "istio-proxy", "install-cni"}
	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}
		for _, container := range pod.Spec.Containers {
			if !slices.Contains(containersToCheck, container.Name) {
				continue
			}
			version, versionErr := getImageVersion(container.Image)
			if versionErr != nil {
				return "", versionErr
			}
			if currentVersion.Equal(&semver.Version{}) {
				currentVersion = version
				continue
			}
			if !currentVersion.Equal(version) {
				return "", fmt.Errorf("image version of Pod %s %s do not match other Pods version %s", pod.Name, version.String(), currentVersion.String())
			}
		}
	}
	if currentVersion.Equal(&semver.Version{}) {
		return "", errors.New("unable to obtain installed Istio image version")
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
	//nolint:mnd // this number needs to be fixed
	if len(matches) < 3 {
		return &semver.Version{}, fmt.Errorf("unable to parse container image reference: %s", image)
	}
	version, err := semver.NewVersion(matches[2])
	if err != nil {
		return &semver.Version{}, err
	}
	noPreleaseVersion, err := version.SetPrerelease("")
	if err != nil {
		return &semver.Version{}, err
	}
	return &noPreleaseVersion, nil
}
