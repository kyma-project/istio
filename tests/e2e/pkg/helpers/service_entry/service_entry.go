package service_entry

import (
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	"istio.io/api/networking/v1alpha3"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"istio.io/client-go/pkg/apis/networking/v1"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
)

func CreateServiceEntry(t *testing.T, name, namespace, host, portProtocol, portResolution string, portNumber uint32) error {
	t.Helper()
	t.Logf("creating service entry %s/%s", namespace, name)

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	serviceEntry := &v1.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha3.ServiceEntry{
			Hosts: []string{host},
			Ports: []*v1alpha3.ServicePort{
				{
					Name:     name,
					Number:   portNumber,
					Protocol: portProtocol,
				},
			},
			Resolution: v1alpha3.ServiceEntry_Resolution(v1alpha3.ServiceEntry_Resolution_value[portResolution]),
		},
	}

	t.Logf("applying service entry %+v", serviceEntry)

	err = r.Create(t.Context(), serviceEntry)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			t.Logf("Failed to create service entry: %v", err)
			return err
		}
		t.Logf("service entry %s/%s already exists", namespace, name)
	} else {
		t.Logf("service entry %s/%s created", namespace, name)
	}

	setup.DeclareCleanup(t, func() {
		err := r.Delete(setup.GetCleanupContext(), serviceEntry)
		if err != nil {
			t.Logf("Failed to delete service entry: %v", err)
		} else {
			t.Logf("service entry %s/%s deleted", namespace, name)
		}

	})

	return nil
}
