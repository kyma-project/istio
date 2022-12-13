package test

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const annotationName = "kubectl.kubernetes.io/restartedAt"

func (s *scenario) aRestartHappens(sidecarImage string) error {
	warnings, err := sidecars.ProxyReset(context.TODO(),
		s.Client,
		pods.SidecarImage{Repository: "istio/proxyv2", Tag: sidecarImage},
		s.injectionNamespaceSelector == SidecarEnabledAndDefault,
		s.cniEnabled,
		&s.logger)
	s.restartWarnings = warnings
	return err
}

func (s *scenario) allRequiredResourcesAreDeleted() error {
	for _, v := range s.ToBeDeletedObjects {
		obj := v
		err := s.Client.Get(context.TODO(), types.NamespacedName{Name: v.GetName(), Namespace: v.GetNamespace()}, obj)
		if err == nil {
			return fmt.Errorf("the Pod %s/%s was deleted but shouldn't", v.GetNamespace(), v.GetName())
		}

		if !k8serrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (s *scenario) allRequiredResourcesAreRestarted() error {
	for _, v := range s.ToBeRestartedObjects {
		obj := v
		err := s.Client.Get(context.TODO(), types.NamespacedName{Name: v.GetName(), Namespace: v.GetNamespace()}, obj)
		if err != nil {
			return err
		}

		if _, ok := obj.GetAnnotations()[annotationName]; !ok {
			return fmt.Errorf("the annotation %s wasn't applied for %s %s/%s", annotationName, obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName())
		}
	}
	return nil
}

func (s *scenario) thereArePodsWithNotYetInjectedSidecars() error {
	return s.WithPodsMissingSidecar()
}

func (s *scenario) WithConfig(istioVersion, injection, cni string) error {
	s.istioVersion = istioVersion
	s.cniEnabled = cni == "true"
	if injection == "true" {
		s.injectionNamespaceSelector = SidecarEnabledAndDefault
	}
	return nil
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	var s scenario

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		scen, err := NewScenario()
		s = *scen
		return ctx, err
	})

	ctx.Step(`^there is a cluster with Istio "([^"]*)", default injection == "([^"]*)" and CNI enabled == "([^"]*)"$`, s.WithConfig)
	ctx.Step(`^a restart happens with target Istio "([^"]*)"`, s.aRestartHappens)
	ctx.Step(`^all required resources are deleted$`, s.allRequiredResourcesAreDeleted)
	ctx.Step(`^all required resources are restarted$`, s.allRequiredResourcesAreRestarted)
	ctx.Step(`^there are Pods missing sidecar`, s.WithPodsMissingSidecar)
	ctx.Step(`^there are not ready Pods$`, s.WithNotReadyPods)
	ctx.Step(`^there are Pods with Istio "([^"]*)" sidecar$`, s.WithSidecarInVersionXPods)
}
