package test

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/test/helpers"
)

const sidecarDisabledNamespace = "sidecar-disabled"
const sidecarEnabledNamespace = "sidecar-enabled"
const noAnnotationNamespace = "default-sidecar"

type NamespaceSelector uint

const (
	NoNamespace                      NamespaceSelector = 0
	Default                          NamespaceSelector = 1
	SidecarDisabled                  NamespaceSelector = 2
	DisabledAndDefault               NamespaceSelector = 3
	SidecarEnabled                   NamespaceSelector = 4
	SidecarEnabledAndDefault         NamespaceSelector = 5
	SidecarEnabledAndSidecarDisabled NamespaceSelector = 6
	AllNamespaces                    NamespaceSelector = 7
)

type scenario struct {
	Client               client.Client
	ToBeDeletedObjects   []client.Object
	ToBeRestartedObjects []client.Object
	logger               logr.Logger
	istioVersion         string
}

func NewScenario() (*scenario, error) {
	err := corev1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	err = appsv1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	return &scenario{
		Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			helpers.FixNamespaceWith(noAnnotationNamespace, nil),
			helpers.FixNamespaceWith(sidecarEnabledNamespace, map[string]string{"istio-injection": "enabled"}),
			helpers.FixNamespaceWith(sidecarDisabledNamespace, map[string]string{"istio-injection": "disabled"}),
		).Build(),
		logger: logr.Discard(),
	}, nil
}

func (b *scenario) WithIstioVersion(istioVersion string) {
	b.istioVersion = istioVersion
}

func (b *scenario) WithNotYetInjectedPods() error {
	notInjected := helpers.NewSidecarPodBuilder().DisableSidecar().SetName("not-injected").Build()
	notInjected.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Deployment",
			Name: "owner",
		},
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "owner",
		},
	}

	err := b.createObjectInAllNamespaces(notInjected, NoNamespace, NoNamespace)
	if err != nil {
		return err
	}

	return b.createObjectInAllNamespaces(&deployment, NoNamespace, SidecarEnabled)
}

func (b *scenario) WithNotReadyPods(t *testing.T, shouldAddNotReadyPods bool) error {
	pendingPod := helpers.NewSidecarPodBuilder().SetPodStatusPhase("Pending").SetName("pending-pod").Build()

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "owner",
		},
	}
	deployment.DeepCopy()

	err := b.createObjectInAllNamespaces(pendingPod, NoNamespace, NoNamespace)
	if err != nil {
		return err
	}

	return b.createObjectInAllNamespaces(&deployment, NoNamespace, SidecarEnabled)
}

func (b *scenario) createObjectInAllNamespaces(toCreate client.Object, deleteIn NamespaceSelector, restartIn NamespaceSelector) error {
	toCreateDefault := helpers.Clone(toCreate).(client.Object)
	toCreateDefault.SetNamespace(noAnnotationNamespace)
	err := b.Client.Create(context.TODO(), toCreateDefault)
	if err != nil {
		return err
	}

	if deleteIn&Default > 0 {
		b.ToBeDeletedObjects = append(b.ToBeDeletedObjects, toCreateDefault)
	}

	if restartIn&Default > 0 {
		b.ToBeRestartedObjects = append(b.ToBeRestartedObjects, toCreateDefault)
	}

	toCreateDisabled := helpers.Clone(toCreate).(client.Object)
	toCreateDisabled.SetNamespace(sidecarDisabledNamespace)
	err = b.Client.Create(context.TODO(), toCreateDisabled)
	if err != nil {
		return err
	}

	if deleteIn&SidecarDisabled > 0 {
		b.ToBeDeletedObjects = append(b.ToBeDeletedObjects, toCreateDisabled)
	}

	if restartIn&SidecarDisabled > 0 {
		b.ToBeRestartedObjects = append(b.ToBeRestartedObjects, toCreateDisabled)
	}

	toCreateEnabled := helpers.Clone(toCreate).(client.Object)
	toCreateEnabled.SetNamespace(sidecarEnabledNamespace)
	err = b.Client.Create(context.TODO(), toCreateEnabled)
	if err != nil {
		return err
	}

	if deleteIn&SidecarEnabled > 0 {
		b.ToBeDeletedObjects = append(b.ToBeDeletedObjects, toCreateEnabled)
	}

	if restartIn&SidecarEnabled > 0 {
		b.ToBeRestartedObjects = append(b.ToBeRestartedObjects, toCreateEnabled)
	}
	return nil
}
