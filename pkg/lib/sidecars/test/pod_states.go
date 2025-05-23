package test

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/test/helpers"
)

func (s *scenario) createObjectInAllNamespaces(toCreate client.Object, deleteIn NamespaceSelector, restartIn NamespaceSelector) error {
	toCreateDefault := helpers.Clone(toCreate).(client.Object)
	toCreateDefault.SetNamespace(noAnnotationNamespace)
	err := s.Client.Create(context.Background(), toCreateDefault)
	if err != nil {
		return err
	}

	if deleteIn&Default > 0 {
		s.ToBeDeletedObjects = append(s.ToBeDeletedObjects, toCreateDefault)
	} else {
		s.NotToBeDeletedObjects = append(s.NotToBeDeletedObjects, toCreateDefault)
	}

	if restartIn&Default > 0 {
		s.ToBeRestartedObjects = append(s.ToBeRestartedObjects, toCreateDefault)
	} else {
		s.NotToBeRestartedObjects = append(s.NotToBeRestartedObjects, toCreateDefault)
	}

	toCreateDisabled := helpers.Clone(toCreate).(client.Object)
	toCreateDisabled.SetNamespace(sidecarDisabledNamespace)
	err = s.Client.Create(context.Background(), toCreateDisabled)
	if err != nil {
		return err
	}

	if deleteIn&SidecarDisabled > 0 {
		s.ToBeDeletedObjects = append(s.ToBeDeletedObjects, toCreateDisabled)
	} else {
		s.NotToBeDeletedObjects = append(s.NotToBeDeletedObjects, toCreateDisabled)
	}

	if restartIn&SidecarDisabled > 0 {
		s.ToBeRestartedObjects = append(s.ToBeRestartedObjects, toCreateDisabled)
	} else {
		s.NotToBeRestartedObjects = append(s.NotToBeRestartedObjects, toCreateDisabled)
	}

	toCreateEnabled := helpers.Clone(toCreate).(client.Object)
	toCreateEnabled.SetNamespace(sidecarEnabledNamespace)
	err = s.Client.Create(context.Background(), toCreateEnabled)
	if err != nil {
		return err
	}

	if deleteIn&SidecarEnabled > 0 {
		s.ToBeDeletedObjects = append(s.ToBeDeletedObjects, toCreateEnabled)
	} else {
		s.NotToBeDeletedObjects = append(s.NotToBeDeletedObjects, toCreateEnabled)
	}

	if restartIn&SidecarEnabled > 0 {
		s.ToBeRestartedObjects = append(s.ToBeRestartedObjects, toCreateEnabled)
	} else {
		s.NotToBeRestartedObjects = append(s.NotToBeRestartedObjects, toCreateEnabled)
	}
	return nil
}

func (s *scenario) WithSidecarInVersionXPods(sidecarTag string) error {
	injectedIstioPod := helpers.NewSidecarPodBuilder().SetSidecarImageTag(sidecarTag).SetName(fmt.Sprintf("injected-%s", sidecarTag)).Build()
	injectedIstioPod.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Deployment",
			Name: fmt.Sprintf("owner-injected-%s", sidecarTag),
		},
	}

	deployment := appsv1.Deployment{
		// The TypeMeta needs to be set due to some recent changes in the fake client as it populates the TypeMeta only for
		// unstructured since version 0.17.0. See: https://github.com/kubernetes-sigs/controller-runtime/pull/2633
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("owner-injected-%s", sidecarTag),
		},
	}

	err := s.createObjectInAllNamespaces(injectedIstioPod, NoNamespace, NoNamespace)
	if err != nil {
		return err
	}

	selector := AllNamespaces
	if sidecarTag == s.istioVersion {
		// We don't support restart in any other case than Istio version change
		selector = NoNamespace
	}

	return s.createObjectInAllNamespaces(&deployment, NoNamespace, selector)
}

func (s *scenario) WithSidecarWithResources(sidecarTag string, resourceType string, cpu string, memory string) error {
	builder := helpers.NewSidecarPodBuilder().
		SetSidecarImageTag(sidecarTag).
		SetName(fmt.Sprintf("injected-%s", sidecarTag))

	switch resourceType {
	case "requests":
		builder.SetCpuRequest(cpu).SetMemoryRequest(memory)
	case "limits":
		builder.SetCpuLimit(cpu).SetMemoryLimit(memory)
	default:
		return fmt.Errorf("unknown resource type %s", resourceType)
	}

	injectedIstioPod := builder.Build()

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

	err := s.createObjectInAllNamespaces(injectedIstioPod, NoNamespace, NoNamespace)
	if err != nil {
		return err
	}

	selector := AllNamespaces
	if sidecarTag == s.istioVersion {
		// We don't support restart in any other case than Istio version change
		selector = NoNamespace
	}

	return s.createObjectInAllNamespaces(&deployment, NoNamespace, selector)
}

func (s *scenario) WithPodsMissingSidecar() error {
	notInjected := helpers.NewSidecarPodBuilder().DisableSidecar().SetName("not-injected").Build()
	notInjected.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Deployment",
			Name: "owner-missing-sidecar",
		},
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "owner-missing-sidecar",
		},
	}

	err := s.createObjectInAllNamespaces(notInjected, NoNamespace, NoNamespace)
	if err != nil {
		return err
	}

	// The restart for pods missing sidecar was implemented as a workaround and will no longer be supported for Istio >= 1.16, so restartIn is set to NoNamespace
	return s.createObjectInAllNamespaces(&deployment, NoNamespace, NoNamespace)
}

func (s *scenario) WithNotReadyPods() error {
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

	err := s.createObjectInAllNamespaces(pendingPod, NoNamespace, NoNamespace)
	if err != nil {
		return err
	}

	return s.createObjectInAllNamespaces(&deployment, NoNamespace, NoNamespace)
}
