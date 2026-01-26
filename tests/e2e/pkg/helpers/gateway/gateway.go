package extauth

import (
	"bytes"
	_ "embed"
	"testing"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/e2e-framework/klient/decoder"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

//go:embed http_gateway.yaml
var httpGateway []byte

// CreateHTTPGateway creates Istio Gateway CR exposing HTTP port 80
// The Gateway matches all hosts ("*")
func CreateHTTPGateway(t *testing.T) error {
	t.Helper()
	t.Log("Creating HTTP Gateway")

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	gw := &unstructured.Unstructured{}
	err = decoder.Decode(
		bytes.NewBuffer(httpGateway),
		gw,
	)
	if err != nil {
		t.Logf("Failed to decode HTTP Gateway template: %v", err)
		return err
	}
	err = r.Create(t.Context(), gw)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			t.Logf("Failed to create HTTP Gateway: %v", err)
			return err
		}
		t.Logf("HTTP Gateway already exists")
	} else {
		t.Logf("HTTP Gateway created")
	}
	setup.DeclareCleanup(t, func() {
		err := r.Delete(setup.GetCleanupContext(), gw)
		if err != nil {
			t.Logf("Failed to delete HTTP Gateway: %v", err)
		} else {
			t.Logf("HTTP Gateway deleted")
		}
	})
	return nil
}
