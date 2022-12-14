package test

import (
	"github.com/go-logr/logr"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
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
	Client                     client.Client
	ToBeDeletedObjects         []client.Object
	ToBeRestartedObjects       []client.Object
	NotToBeDeletedObjects      []client.Object
	NotToBeRestartedObjects    []client.Object
	logger                     logr.Logger
	istioVersion               string
	cniEnabled                 bool
	injectionNamespaceSelector NamespaceSelector
	restartWarnings            []restart.RestartWarning
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
		logger:                     logr.Discard(),
		injectionNamespaceSelector: SidecarEnabled,
	}, nil
}
