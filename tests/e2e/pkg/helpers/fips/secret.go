package fips

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
)

const (
	dockerServer   = "europe-docker.pkg.dev"
	dockerUsername = "_json_key"
)

// EnsureFIPSRegistrySecret creates a docker-registry secret in the given namespace
// when KYMA_FIPS_MODE_ENABLED is "true". Creates the namespace if it does not exist.
func EnsureFIPSRegistrySecret(t *testing.T, namespace string) {
	t.Helper()

	if os.Getenv("KYMA_FIPS_MODE_ENABLED") != "true" {
		t.Log("KYMA_FIPS_MODE_ENABLED is not set to true, skipping FIPS registry secret creation")
		return
	}

	secretName := os.Getenv("REGISTRY_SECRET_NAME")
	if secretName == "" {
		secretName = os.Getenv("SKR_IMG_PULL_SECRET")
	}
	if secretName == "" {
		t.Fatal("REGISTRY_SECRET_NAME or SKR_IMG_PULL_SECRET environment variable is required when KYMA_FIPS_MODE_ENABLED is true")
	}

	registrySA := os.Getenv("FIPS_REGISTRY_SA")
	if registrySA == "" {
		t.Fatal("FIPS_REGISTRY_SA environment variable is required when KYMA_FIPS_MODE_ENABLED is true")
	}

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Fatalf("Failed to create resources client: %v", err)
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	err = r.Create(t.Context(), ns)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			t.Fatalf("Failed to create namespace %s: %v", namespace, err)
		}
		existing := &corev1.Namespace{}
		if getErr := r.Get(t.Context(), namespace, "", existing); getErr == nil && existing.Status.Phase == corev1.NamespaceTerminating {
			t.Logf("Namespace %s is terminating, waiting for deletion", namespace)
			if waitErr := wait.For(conditions.New(r).ResourceDeleted(existing), wait.WithTimeout(2*time.Minute)); waitErr != nil {
				t.Fatalf("Timed out waiting for terminating namespace %s to be deleted: %v", namespace, waitErr)
			}
			if createErr := r.Create(t.Context(), ns); createErr != nil {
				t.Fatalf("Failed to create namespace %s after waiting for deletion: %v", namespace, createErr)
			}
		}
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: buildDockerConfigJSON(dockerServer, dockerUsername, registrySA),
		},
	}

	err = r.Create(t.Context(), secret)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			t.Logf("FIPS registry secret %s already exists in %s, updating it", secretName, namespace)
			err = r.Update(t.Context(), secret)
			if err != nil {
				t.Fatalf("Failed to update FIPS registry secret: %v", err)
			}
		} else {
			t.Fatalf("Failed to create FIPS registry secret: %v", err)
		}
	}

	t.Logf("FIPS registry secret %s created/updated in namespace %s", secretName, namespace)
}

func buildDockerConfigJSON(server, username, password string) []byte {
	dockerConfig := map[string]interface{}{
		"auths": map[string]interface{}{
			server: map[string]string{
				"username": username,
				"password": password,
			},
		},
	}
	data, _ := json.Marshal(dockerConfig)
	return data
}
