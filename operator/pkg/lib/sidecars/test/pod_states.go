package test

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/test/helpers"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func (b *scenario) WithSidecarInVersionXPods(sidecarTag string) error {
	injectedIstioPod := helpers.NewSidecarPodBuilder().SetSidecarImageTag(sidecarTag).SetName(fmt.Sprintf("injected-%s", sidecarTag)).Build()
	injectedIstioPod.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Deployment",
			Name: fmt.Sprintf("owner-injected-%s", sidecarTag),
		},
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("owner-injected-%s", sidecarTag),
		},
	}

	err := b.createObjectInAllNamespaces(injectedIstioPod, NoNamespace, NoNamespace)
	if err != nil {
		return err
	}

	selector := AllNamespaces
	if sidecarTag == b.istioVersion {
		selector = AllNamespaces &^ b.injectionNamespaceSelector
	}

	return b.createObjectInAllNamespaces(&deployment, NoNamespace, selector)
}

func (b *scenario) WithPodsMissingSidecar() error {
	notInjected := helpers.NewSidecarPodBuilder().DisableSidecar().SetName("not-injected").Build()
	notInjected.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Deployment",
			Name: "owner-not-yet",
		},
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "owner-not-yet",
		},
	}

	err := b.createObjectInAllNamespaces(notInjected, NoNamespace, NoNamespace)
	if err != nil {
		return err
	}

	return b.createObjectInAllNamespaces(&deployment, NoNamespace, b.injectionNamespaceSelector)
}

func (b *scenario) WithNotReadyPods() error {
	pendingPod := helpers.NewSidecarPodBuilder().DisableSidecar().SetPodStatusPhase("Pending").SetName("pending-pod").Build()
	pendingPod.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Deployment",
			Name: "owner-not-ready",
		},
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "owner-not-ready",
		},
	}

	err := b.createObjectInAllNamespaces(pendingPod, NoNamespace, NoNamespace)
	if err != nil {
		return err
	}

	return b.createObjectInAllNamespaces(&deployment, NoNamespace, NoNamespace)
}
