package virtual_service

import (
	"testing"

	alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

func CreateVirtualService(t *testing.T, name, namespace, dstHost string, hosts, gateways []string) error {
	t.Helper()
	t.Logf("creating virtual service %s/%s", namespace, name)

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	vs := &v1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: alpha3.VirtualService{
			Gateways: gateways,
			Hosts:    hosts,
			Http: []*alpha3.HTTPRoute{
				{
					Match: []*alpha3.HTTPMatchRequest{
						{
							Uri: &alpha3.StringMatch{
								MatchType: &alpha3.StringMatch_Prefix{
									Prefix: "/",
								},
							},
						},
					},
					Route: []*alpha3.HTTPRouteDestination{
						{
							Destination: &alpha3.Destination{
								Host: dstHost,
							},
						},
					},
				},
			},
		},
	}

	err = r.Create(t.Context(), vs)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			t.Logf("Failed to create virtual service: %v", err)
			return err
		}
		t.Logf("virtual service %s/%s already exists", namespace, name)
	} else {
		t.Logf("virtual service %s/%s created", namespace, name)
	}

	setup.DeclareCleanup(t, func() {
		err := r.Delete(setup.GetCleanupContext(), vs)
		if err != nil {
			t.Logf("Failed to delete virtual service: %v", err)
		} else {
			t.Logf("virtual service %s/%s deleted", namespace, name)
		}

	})

	return nil
}
