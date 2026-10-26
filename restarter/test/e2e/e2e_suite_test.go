//go:build e2e
// +build e2e

/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/kyma-project/istio/operator/workload-controller/test/utils"
)

// managerImage is the manager image to be built and loaded for testing.
var managerImage = "example.com/restarter:v0.0.1"

// shouldCleanupCertManager tracks whether CertManager was installed by this suite.
var shouldCleanupCertManager = false

// istioCRDPath is the path (relative to project root) of the Istio CRD YAML
// that the restarter reads evaluators from. It lives in the parent istio
// operator module and is applied into the Kind cluster before tests run.
const istioCRDPath = "../config/crd/bases/operator.kyma-project.io_istios.yaml"

// TestMain runs the suite-wide setup/teardown that previously lived in
// BeforeSuite / AfterSuite.
//
// Env:
//
//	KUBECTL_KUBERC=true           — keep local kuberc (default: disabled for test isolation)
//	CERT_MANAGER_INSTALL_SKIP=true — skip CertManager install
func TestMain(m *testing.M) {
	if err := suiteSetup(); err != nil {
		fmt.Fprintf(os.Stderr, "suite setup failed: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	suiteTeardown()
	os.Exit(code)
}

func suiteSetup() error {
	fmt.Fprintln(os.Stderr, "Starting restarter e2e test suite")

	fmt.Fprintln(os.Stderr, "building the manager image")
	cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", managerImage))
	if _, err := utils.Run(cmd); err != nil {
		return fmt.Errorf("failed to build the manager image: %w", err)
	}

	// TODO(user): If you want to change the e2e test vendor from Kind,
	// ensure the image is built and available, then remove the following block.
	fmt.Fprintln(os.Stderr, "loading the manager image on Kind")
	if err := utils.LoadImageToKindClusterWithName(managerImage); err != nil {
		return fmt.Errorf("failed to load the manager image into Kind: %w", err)
	}

	configureKubectlKubeRC()

	if err := setupCertManager(); err != nil {
		return err
	}

	return installIstioCRD()
}

func suiteTeardown() {
	uninstallIstioCRD()
	teardownCertManager()
}

// installIstioCRD applies the parent istio operator's Istio CRD so the
// restarter's evaluation.Provider can read the Istio CR at kyma-system/default.
// Without this CRD every reconcile would error on the Get call before reaching
// the evaluator path, defeating the point of the workload-reconciliation tests.
func installIstioCRD() error {
	path, err := filepath.Abs(istioCRDPath)
	if err != nil {
		return fmt.Errorf("resolve istio CRD path: %w", err)
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("istio CRD not found at %s: %w (does the parent istio operator repo live alongside restarter?)", path, err)
	}
	fmt.Fprintln(os.Stderr, "installing Istio CRD from parent operator module")
	cmd := exec.Command("kubectl", "apply", "-f", path)
	if _, err := utils.Run(cmd); err != nil {
		return fmt.Errorf("failed to apply Istio CRD: %w", err)
	}
	return nil
}

func uninstallIstioCRD() {
	path, err := filepath.Abs(istioCRDPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve istio CRD path: %v\n", err)
		return
	}
	fmt.Fprintln(os.Stderr, "uninstalling Istio CRD")
	cmd := exec.Command("kubectl", "delete", "-f", path, "--ignore-not-found")
	if _, err := utils.Run(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to delete Istio CRD: %v\n", err)
	}
}

// Disable kubectl kuberc by default for test isolation.
// This prevents local kubectl configurations from affecting test behavior.
// To enable kuberc, set: KUBECTL_KUBERC=true
func configureKubectlKubeRC() {
	if os.Getenv("KUBECTL_KUBERC") != "true" {
		fmt.Fprintln(os.Stderr, "disabling kubectl kuberc for test isolation")
		if err := os.Setenv("KUBECTL_KUBERC", "false"); err != nil {
			fmt.Fprintf(os.Stderr, "failed to disable kubectl kuberc: %v\n", err)
		}
		fmt.Fprintln(os.Stderr,
			"kubectl kuberc disabled for consistent test behavior (override with KUBECTL_KUBERC=true)")
	} else {
		fmt.Fprintln(os.Stderr, "kubectl kuberc enabled (KUBECTL_KUBERC=true)")
	}
}

// setupCertManager installs CertManager if needed for webhook tests.
// Skips installation if CERT_MANAGER_INSTALL_SKIP=true or if already present.
func setupCertManager() error {
	if os.Getenv("CERT_MANAGER_INSTALL_SKIP") == "true" {
		fmt.Fprintln(os.Stderr, "Skipping CertManager installation (CERT_MANAGER_INSTALL_SKIP=true)")
		return nil
	}

	fmt.Fprintln(os.Stderr, "checking if CertManager is already installed")
	if utils.IsCertManagerCRDsInstalled() {
		fmt.Fprintln(os.Stderr, "CertManager is already installed. Skipping installation.")
		return nil
	}

	// Mark for cleanup before installation to handle interruptions and partial installs.
	shouldCleanupCertManager = true

	fmt.Fprintln(os.Stderr, "installing CertManager")
	if err := utils.InstallCertManager(); err != nil {
		return fmt.Errorf("failed to install CertManager: %w", err)
	}
	return nil
}

// teardownCertManager uninstalls CertManager if it was installed by setupCertManager.
// This ensures we only remove what we installed.
func teardownCertManager() {
	if !shouldCleanupCertManager {
		fmt.Fprintln(os.Stderr, "Skipping CertManager cleanup (not installed by this suite)")
		return
	}

	fmt.Fprintln(os.Stderr, "uninstalling CertManager")
	utils.UninstallCertManager()
}

// pollUntil repeatedly calls fn until it returns nil or timeout elapses.
// Mirrors the Ginkgo Eventually(...).Should(Succeed()) semantics.
func pollUntil(t *testing.T, timeout, interval time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		lastErr = fn()
		if lastErr == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out after %s: %v", timeout, lastErr)
		}
		time.Sleep(interval)
	}
}
