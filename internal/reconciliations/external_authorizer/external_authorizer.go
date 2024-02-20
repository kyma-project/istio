package external_authorizer

import (
	"context"
	"fmt"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	moduleLabels "github.com/kyma-project/istio/operator/pkg/labels"
	apinetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	ServiceEntryNamespace = "kyma-system"

	ServiceEntryApiVersion = "networking.istio.io/v1alpha3"
	ServiceEntryKind       = "ServiceEntry"

	HttpProtocol = "http"
)

type ExternalAuthorizerReconciliation interface {
	Reconcile(ctx context.Context, istioCR operatorv1alpha2.Istio) described_errors.DescribedError
}

type ExternalAuthorizer struct {
	client client.Client
}

func NewReconciler(k8sClient client.Client) *ExternalAuthorizer {
	return &ExternalAuthorizer{client: k8sClient}
}

func (e *ExternalAuthorizer) Reconcile(ctx context.Context, istioCR operatorv1alpha2.Istio) described_errors.DescribedError {
	authorizersNameSet := make(map[string]bool)
	for _, authorizer := range istioCR.Spec.Config.Authorizers {
		_, exists := authorizersNameSet[authorizer.Name]
		if exists {
			return described_errors.NewDescribedError(fmt.Errorf("%s is dupplicated", authorizer.Name), "Authorizer name needs to be unique")
		}
		authorizersNameSet[authorizer.Name] = true
	}

	for _, authorizer := range istioCR.Spec.Config.Authorizers {
		err := applyServiceEntryForAuthorizer(ctx, e.client, authorizer)
		if err != nil {
			return described_errors.NewDescribedError(err, fmt.Sprintf("Could not succesfully apply authorizer %s", authorizer.Name))
		}
	}

	return cleanupServiceEntries(ctx, e.client, authorizersNameSet)
}

func cleanupServiceEntries(ctx context.Context, c client.Client, authorizersNameSet map[string]bool) described_errors.DescribedError {
	var serviceEntryList unstructured.UnstructuredList

	serviceEntryList.SetKind(ServiceEntryKind)
	serviceEntryList.SetAPIVersion(ServiceEntryApiVersion)
	err := c.List(ctx, &serviceEntryList)
	if err != nil {
		return described_errors.NewDescribedError(err, "Could not list ServiceEntries")
	}

	for _, serviceEntry := range serviceEntryList.Items {
		_, exists := authorizersNameSet[serviceEntry.GetName()]
		if !exists {
			val, exists := serviceEntry.GetLabels()[moduleLabels.ModuleLabelKey]
			if exists && val == moduleLabels.ModuleLabelValue {
				err = c.Delete(ctx, &serviceEntry)
				if err != nil {
					return described_errors.NewDescribedError(err, "Could not delete ServiceEntry that was removed from authorizers")
				}
			}
		}
	}

	return nil
}

func applyServiceEntryForAuthorizer(ctx context.Context, k8sClient client.Client, authorizer *operatorv1alpha2.Authorizer) error {
	if authorizer == nil {
		return nil
	}

	serviceEntry := v1alpha3.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      authorizer.Name,
			Namespace: ServiceEntryNamespace,
			Labels: map[string]string{
				moduleLabels.ModuleLabelKey: moduleLabels.ModuleLabelValue,
			},
		},
		Spec: apinetworkingv1alpha3.ServiceEntry{
			Hosts: []string{authorizer.Service},
			Ports: []*apinetworkingv1alpha3.ServicePort{
				{
					Name:     HttpProtocol,
					Number:   authorizer.Port,
					Protocol: HttpProtocol,
				},
			},
			Resolution: apinetworkingv1alpha3.ServiceEntry_STATIC,
		},
	}

	serviceEntryMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&serviceEntry)
	serviceEntryUnstructured := unstructured.Unstructured{Object: serviceEntryMap}
	serviceEntryUnstructured.SetKind(ServiceEntryKind)
	serviceEntryUnstructured.SetAPIVersion(ServiceEntryApiVersion)

	spec, specExist := serviceEntryUnstructured.Object["spec"]
	labels := serviceEntryUnstructured.GetLabels()
	_, err = controllerutil.CreateOrUpdate(ctx, k8sClient, &serviceEntryUnstructured, func() error {
		if len(labels) == 0 {
			labels = map[string]string{}
		}
		serviceEntryUnstructured.SetLabels(labels)

		if specExist {
			serviceEntryUnstructured.Object["spec"] = spec
		}

		return nil
	})
	return err
}
