package telemetry

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/e2e-framework/klient/decoder"
)

//go:embed telemetry_traces.yaml
var tracesCR []byte

func EnableTraces(t *testing.T) error {
	t.Helper()
	t.Log("Creating Telemetry Resource with global logs enabled")

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	tm := &unstructured.Unstructured{}
	err = decoder.Decode(
		bytes.NewBuffer(tracesCR),
		tm,
	)

	if err != nil {
		t.Logf("Failed to decode telemetry CR: %v", err)
		return err
	}

	err = r.Create(t.Context(), tm)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			t.Logf("Failed to create Telemetry resource: %v", err)
			return err
		}
		t.Logf("Telemetry resource already exists: %v", tm)
	} else {
		t.Logf("Telemetry resource created")
	}
	setup.DeclareCleanup(t, func() {
		err := r.Delete(setup.GetCleanupContext(), tm)
		if err != nil {
			t.Logf("Failed to delete Telemetry resource: %v", err)
		} else {
			t.Logf("Telemetry resource deleted")
		}
	})
	return nil
}
