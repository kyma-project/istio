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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/istio/operator/workload-controller/test/utils"
)

// namespace where the project is deployed in
const namespace = "restarter-system"

// serviceAccountName created for the project
const serviceAccountName = "restarter-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "restarter-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "restarter-metrics-binding"

// istioCRNamespace is where the restarter expects the Istio CR to live.
// Matches the default --istio-cr-namespace flag in cmd/main.go.
const istioCRNamespace = "kyma-system"

// istioCRName matches the default --istio-cr-name flag in cmd/main.go.
const istioCRName = "default"

// expectedSidecarImage is the proxyv2 image the restarter is configured with
// via config/manager/env-images.yaml. Pods whose istio-proxy container uses
// this exact string are considered current; any other value trips a Restart.
const expectedSidecarImage = "europe-docker.pkg.dev/kyma-project/prod/external/istio/proxyv2:1.31.1-distroless"

// outdatedSidecarImage is any value distinct from expectedSidecarImage. The
// image never pulls — the restarter inspects spec, not status, so an invalid
// reference is fine.
const outdatedSidecarImage = "istio/proxyv2:0.0.0-outdated"

// appContainerImage is the non-sidecar container image used by test workload
// pods. pause is tiny and already pre-pulled in Kind nodes.
const appContainerImage = "registry.k8s.io/pause:3.9"

// defaultTimeout bounds event-visibility polls.
const defaultTimeout = 2 * time.Minute

// defaultInterval is the poll cadence inside pollUntil.
const defaultInterval = time.Second

// TestManager runs the ordered e2e scenarios against the deployed controller.
// Subtests share state (controllerPodName) via closure and run sequentially via
// explicit t.Run calls, mirroring the original Ginkgo `Ordered` container.
//
// Scenarios in order:
//  1. controller pod running
//  2. metrics endpoint serves 200 OK
//  3. no Istio CR present → no events emitted for outdated workloads
//  4. Deployment with outdated sidecar → RestartRequired event
//  5. StatefulSet with current sidecar → no RestartRequired event
//  6. DaemonSet with outdated sidecar → RestartRequired event (field indexer path)
//  7. bare ReplicaSet gets event, Deployment-owned ReplicaSet does not
//  8. workloads in two non-system namespaces both get events
//  9. controller-runtime reconcile counters increment per workload kind
func TestManager(t *testing.T) {
	t.Log("creating manager namespace")
	cmd := exec.Command("kubectl", "create", "ns", namespace)
	if _, err := utils.Run(cmd); err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}

	t.Log("labeling the namespace to enforce the restricted security policy")
	cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
		"pod-security.kubernetes.io/enforce=restricted")
	if _, err := utils.Run(cmd); err != nil {
		t.Fatalf("failed to label namespace with restricted policy: %v", err)
	}

	t.Log("installing CRDs")
	cmd = exec.Command("make", "install")
	if _, err := utils.Run(cmd); err != nil {
		t.Fatalf("failed to install CRDs: %v", err)
	}

	t.Log("deploying the controller-manager")
	cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", managerImage))
	if _, err := utils.Run(cmd); err != nil {
		t.Fatalf("failed to deploy the controller-manager: %v", err)
	}

	t.Cleanup(func() {
		t.Log("cleaning up the curl pod for metrics")
		_, _ = utils.Run(exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace))

		t.Log("cleaning up metrics ClusterRoleBinding")
		_, _ = utils.Run(exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName, "--ignore-not-found"))

		t.Log("undeploying the controller-manager")
		_, _ = utils.Run(exec.Command("make", "undeploy"))

		t.Log("uninstalling CRDs")
		_, _ = utils.Run(exec.Command("make", "uninstall"))

		t.Log("removing manager namespace")
		_, _ = utils.Run(exec.Command("kubectl", "delete", "ns", namespace))
	})

	// controllerPodName captures the deployed controller pod identity in the
	// first subtest and is read by later subtests' failureDiag for log dumps.
	// Shared via closure, not package scope — a future t.Parallel() would then
	// trigger the race detector at the use site.
	var controllerPodName string

	// kymaSystemCreated tracks whether ensureIstioCR had to create the
	// kyma-system namespace. We delete it on teardown only if we created it,
	// mirroring shouldCleanupCertManager in e2e_suite_test.go. Otherwise a
	// developer cluster that already had kyma-system (e.g. from running the
	// Istio operator) would be wiped by the test.
	var kymaSystemCreated bool

	t.Run("controller pod runs", func(t *testing.T) {
		testControllerPodRunning(t, &controllerPodName)
	})
	t.Run("metrics endpoint serves", func(t *testing.T) {
		testMetricsEndpoint(t, controllerPodName)
	})
	t.Run("missing Istio CR emits no events", func(t *testing.T) {
		testNoIstioCRNoEvents(t, controllerPodName)
	})

	t.Log("creating Istio CR for evaluator-dependent scenarios")
	kymaSystemCreated = ensureIstioCR(t)
	t.Cleanup(func() {
		t.Log("deleting Istio CR")
		_, _ = utils.Run(exec.Command("kubectl", "delete", "istio", istioCRName,
			"-n", istioCRNamespace, "--ignore-not-found"))
		if kymaSystemCreated {
			t.Log("deleting kyma-system namespace (created by this suite)")
			_, _ = utils.Run(exec.Command("kubectl", "delete", "ns", istioCRNamespace, "--ignore-not-found"))
		} else {
			t.Log("preserving pre-existing kyma-system namespace")
		}
	})

	t.Run("Deployment with outdated sidecar emits RestartRequired", func(t *testing.T) {
		testDeploymentOutdatedEmitsEvent(t, controllerPodName)
	})
	t.Run("StatefulSet with current sidecar emits no RestartRequired", func(t *testing.T) {
		testStatefulSetCurrentNoEvent(t, controllerPodName)
	})
	t.Run("DaemonSet with outdated sidecar emits RestartRequired", func(t *testing.T) {
		testDaemonSetOutdatedEmitsEvent(t, controllerPodName)
	})
	t.Run("bare ReplicaSet evaluated, owned ReplicaSet skipped", func(t *testing.T) {
		testReplicaSetOwnershipFilter(t, controllerPodName)
	})
	t.Run("workloads across namespaces both emit events", func(t *testing.T) {
		testCrossNamespaceEvents(t, controllerPodName)
	})
	t.Run("reconcile counters increment per workload kind", func(t *testing.T) {
		testReconcileMetricsPerKind(t, controllerPodName)
	})
}

// failureDiag dumps controller logs and events on subtest failure. Registered
// per subtest so only the failing scenario's context is dumped.
func failureDiag(t *testing.T, controllerPodName string) {
	t.Helper()
	if !t.Failed() {
		return
	}
	t.Log("Fetching controller manager pod logs")
	if controllerPodName == "" {
		t.Log("controller pod name unknown (first subtest failed before capture)")
	} else if out, err := utils.Run(exec.Command("kubectl", "logs", controllerPodName, "-n", namespace, "--tail=200")); err == nil {
		t.Logf("Controller logs:\n %s", out)
	} else {
		t.Logf("Failed to get Controller logs: %v", err)
	}

	t.Log("Fetching Kubernetes events in restarter-system")
	if out, err := utils.Run(exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")); err == nil {
		t.Logf("Kubernetes events:\n%s", out)
	} else {
		t.Logf("Failed to get Kubernetes events: %v", err)
	}
}

func testControllerPodRunning(t *testing.T, controllerPodName *string) {
	t.Cleanup(func() { failureDiag(t, *controllerPodName) })

	t.Log("validating that the controller-manager pod is running as expected")
	pollUntil(t, defaultTimeout, defaultInterval, func() error {
		cmd := exec.Command("kubectl", "get",
			"pods", "-l", "control-plane=controller-manager",
			"-o", "go-template={{ range .items }}"+
				"{{ if not .metadata.deletionTimestamp }}"+
				"{{ .metadata.name }}"+
				"{{ \"\\n\" }}{{ end }}{{ end }}",
			"-n", namespace,
		)
		podOutput, err := utils.Run(cmd)
		if err != nil {
			return fmt.Errorf("failed to retrieve controller-manager pod information: %w", err)
		}
		podNames := utils.GetNonEmptyLines(podOutput)
		if len(podNames) != 1 {
			return fmt.Errorf("expected 1 controller pod running, got %d", len(podNames))
		}
		*controllerPodName = podNames[0]
		if !strings.Contains(*controllerPodName, "controller-manager") {
			return fmt.Errorf("unexpected pod name: %q", *controllerPodName)
		}

		cmd = exec.Command("kubectl", "get",
			"pods", *controllerPodName, "-o", "jsonpath={.status.phase}",
			"-n", namespace,
		)
		output, err := utils.Run(cmd)
		if err != nil {
			return err
		}
		if output != "Running" {
			return fmt.Errorf("incorrect controller-manager pod status: %q", output)
		}
		return nil
	})
}

func testMetricsEndpoint(t *testing.T, controllerPodName string) {
	t.Cleanup(func() { failureDiag(t, controllerPodName) })

	t.Log("creating a ClusterRoleBinding for the service account to allow access to metrics")
	cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
		"--clusterrole=restarter-metrics-reader",
		fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
	)
	if _, err := utils.Run(cmd); err != nil {
		t.Fatalf("failed to create ClusterRoleBinding: %v", err)
	}

	t.Log("validating that the metrics service is available")
	cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
	if _, err := utils.Run(cmd); err != nil {
		t.Fatalf("metrics service should exist: %v", err)
	}

	t.Log("getting the service account token")
	token, err := serviceAccountToken(t)
	if err != nil {
		t.Fatalf("getting service account token: %v", err)
	}
	if token == "" {
		t.Fatal("service account token is empty")
	}

	t.Log("ensuring the controller pod is ready")
	pollUntil(t, 3*time.Minute, time.Second, func() error {
		cmd := exec.Command("kubectl", "get", "pod", controllerPodName, "-n", namespace,
			"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
		output, err := utils.Run(cmd)
		if err != nil {
			return err
		}
		if output != "True" {
			return fmt.Errorf("controller pod not ready: %q", output)
		}
		return nil
	})

	t.Log("verifying that the controller manager is serving the metrics server")
	pollUntil(t, 3*time.Minute, time.Second, func() error {
		cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
		output, err := utils.Run(cmd)
		if err != nil {
			return err
		}
		if !strings.Contains(output, "Serving metrics server") {
			return fmt.Errorf("metrics server not yet started")
		}
		return nil
	})

	t.Log("creating the curl-metrics pod to access the metrics endpoint")
	runCurlMetrics(t, token, true)

	t.Log("waiting for the curl-metrics pod to complete.")
	pollUntil(t, 5*time.Minute, time.Second, func() error {
		cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
			"-o", "jsonpath={.status.phase}",
			"-n", namespace)
		output, err := utils.Run(cmd)
		if err != nil {
			return err
		}
		if output != "Succeeded" {
			return fmt.Errorf("curl pod in wrong status: %q", output)
		}
		return nil
	})

	t.Log("getting the metrics by checking curl-metrics logs")
	pollUntil(t, 2*time.Minute, time.Second, func() error {
		metricsOutput, err := getMetricsOutput()
		if err != nil {
			return fmt.Errorf("failed to retrieve logs from curl pod: %w", err)
		}
		if metricsOutput == "" {
			return fmt.Errorf("metrics output is empty")
		}
		if !strings.Contains(metricsOutput, "< HTTP/1.1 200 OK") {
			return fmt.Errorf("metrics endpoint did not return 200 OK:\n%s", metricsOutput)
		}
		return nil
	})
}

// testNoIstioCRNoEvents verifies the Snapshot(nil, nil) code path. Without
// an Istio CR, no evaluator runs; even outdated sidecars must not trigger
// RestartRequired. Runs before ensureIstioCR.
func testNoIstioCRNoEvents(t *testing.T, controllerPodName string) {
	t.Cleanup(func() { failureDiag(t, controllerPodName) })

	ns := createWorkloadNamespace(t, "nocr")
	applyManifest(t, deploymentWithSidecar("no-cr-dep", ns, outdatedSidecarImage))

	t.Log("confirming the deployment pod comes up before asserting negative event state")
	waitForOnePod(t, ns, "app=no-cr-dep")

	// Give the controller several reconcile cycles to vote and still emit
	// nothing. 30s is well above typical reconcile latency.
	t.Log("sleeping to allow the controller to attempt and skip reconciliation")
	time.Sleep(30 * time.Second)

	assertNoEventWithReason(t, ns, "Deployment", "no-cr-dep", "RestartRequired")
}

func testDeploymentOutdatedEmitsEvent(t *testing.T, controllerPodName string) {
	t.Cleanup(func() { failureDiag(t, controllerPodName) })

	ns := createWorkloadNamespace(t, "dep-outdated")
	applyManifest(t, deploymentWithSidecar("outdated-dep", ns, outdatedSidecarImage))

	waitForEventReason(t, ns, "Deployment", "outdated-dep", "RestartRequired")
}

func testStatefulSetCurrentNoEvent(t *testing.T, controllerPodName string) {
	t.Cleanup(func() { failureDiag(t, controllerPodName) })

	ns := createWorkloadNamespace(t, "sts-current")
	applyManifest(t, statefulSetWithSidecar("current-sts", ns, expectedSidecarImage))

	waitForOnePod(t, ns, "app=current-sts")

	t.Log("sleeping to allow the controller to attempt and skip reconciliation")
	time.Sleep(30 * time.Second)

	assertNoEventWithReason(t, ns, "StatefulSet", "current-sts", "RestartRequired")
}

// testDaemonSetOutdatedEmitsEvent exercises the DaemonSet field-indexer path:
// pods are listed by owner UID, not label selector, so this covers a different
// code path from the Deployment case.
func testDaemonSetOutdatedEmitsEvent(t *testing.T, controllerPodName string) {
	t.Cleanup(func() { failureDiag(t, controllerPodName) })

	ns := createWorkloadNamespace(t, "ds-outdated")
	applyManifest(t, daemonSetWithSidecar("outdated-ds", ns, outdatedSidecarImage))

	waitForEventReason(t, ns, "DaemonSet", "outdated-ds", "RestartRequired")
}

// testReplicaSetOwnershipFilter verifies the "skip owned ReplicaSet" guard in
// replicaset_controller.go. A bare RS and a Deployment-owned RS both carry
// outdated sidecars; only the bare RS and its parent Deployment emit events.
func testReplicaSetOwnershipFilter(t *testing.T, controllerPodName string) {
	t.Cleanup(func() { failureDiag(t, controllerPodName) })

	ns := createWorkloadNamespace(t, "rs-ownership")
	applyManifest(t, replicaSetWithSidecar("bare-rs", ns, outdatedSidecarImage))
	applyManifest(t, deploymentWithSidecar("owner-dep", ns, outdatedSidecarImage))

	waitForEventReason(t, ns, "ReplicaSet", "bare-rs", "RestartRequired")
	waitForEventReason(t, ns, "Deployment", "owner-dep", "RestartRequired")

	t.Log("looking up the Deployment-owned ReplicaSet")
	var ownedRSName string
	pollUntil(t, 1*time.Minute, defaultInterval, func() error {
		out, err := utils.Run(exec.Command("kubectl", "get", "rs", "-n", ns,
			"-l", "app=owner-dep",
			"-o", "jsonpath={.items[0].metadata.name}"))
		if err != nil {
			return err
		}
		if out == "" {
			return fmt.Errorf("owned RS not yet created")
		}
		ownedRSName = out
		return nil
	})

	time.Sleep(30 * time.Second)
	assertNoEventWithReason(t, ns, "ReplicaSet", ownedRSName, "RestartRequired")
}

func testCrossNamespaceEvents(t *testing.T, controllerPodName string) {
	t.Cleanup(func() { failureDiag(t, controllerPodName) })

	nsA := createWorkloadNamespace(t, "cross-a")
	nsB := createWorkloadNamespace(t, "cross-b")

	applyManifest(t, deploymentWithSidecar("cross-dep", nsA, outdatedSidecarImage))
	applyManifest(t, deploymentWithSidecar("cross-dep", nsB, outdatedSidecarImage))

	waitForEventReason(t, nsA, "Deployment", "cross-dep", "RestartRequired")
	waitForEventReason(t, nsB, "Deployment", "cross-dep", "RestartRequired")
}

func testReconcileMetricsPerKind(t *testing.T, controllerPodName string) {
	t.Cleanup(func() { failureDiag(t, controllerPodName) })

	t.Log("re-running curl-metrics to pull the latest metrics snapshot")
	_, _ = utils.Run(exec.Command("kubectl", "delete", "pod", "curl-metrics",
		"-n", namespace, "--ignore-not-found"))

	token, err := serviceAccountToken(t)
	if err != nil {
		t.Fatalf("getting service account token: %v", err)
	}

	runCurlMetrics(t, token, false)

	pollUntil(t, 2*time.Minute, defaultInterval, func() error {
		out, err := utils.Run(exec.Command("kubectl", "get", "pod", "curl-metrics",
			"-n", namespace, "-o", "jsonpath={.status.phase}"))
		if err != nil {
			return err
		}
		if out != "Succeeded" {
			return fmt.Errorf("curl pod in phase %q", out)
		}
		return nil
	})

	metrics, err := getMetricsOutput()
	if err != nil {
		t.Fatalf("failed to read metrics output: %v", err)
	}

	for _, kind := range []string{"deployment", "replicaset", "statefulset", "daemonset"} {
		needle := fmt.Sprintf(`controller_runtime_reconcile_total{controller="%s"`, kind)
		if !strings.Contains(metrics, needle) {
			t.Errorf("metrics missing counter for controller %q (needle %q)", kind, needle)
		}
	}
}

// runCurlMetrics spawns a one-shot curl pod against the controller /metrics
// endpoint. verbose=true emits `-v` so the metrics-endpoint subtest can parse
// `HTTP/1.1 200 OK` from the headers.
func runCurlMetrics(t *testing.T, token string, verbose bool) {
	t.Helper()
	curlFlags := "-sS"
	if verbose {
		curlFlags = "-v"
	}
	cmd := exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
		"--namespace", namespace,
		"--image=curlimages/curl:latest",
		"--overrides",
		fmt.Sprintf(`{
				"spec": {
					"containers": [{
						"name": "curl",
						"image": "curlimages/curl:latest",
						"command": ["/bin/sh", "-c"],
						"args": [
							"for i in $(seq 1 30); do curl %s -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics && exit 0 || sleep 2; done; exit 1"
						],
						"securityContext": {
							"readOnlyRootFilesystem": true,
							"allowPrivilegeEscalation": false,
							"capabilities": {"drop": ["ALL"]},
							"runAsNonRoot": true,
							"runAsUser": 1000,
							"seccompProfile": {"type": "RuntimeDefault"}
						}
					}],
					"serviceAccountName": "%s"
				}
			}`, curlFlags, token, metricsServiceName, namespace, serviceAccountName))
	if _, err := utils.Run(cmd); err != nil {
		t.Fatalf("failed to create curl-metrics pod: %v", err)
	}
}

// ensureIstioCR applies the kyma-system namespace (if missing) and a bare
// Istio CR. Only existence matters — the evaluation provider short-circuits
// on NotFound, so a minimal metadata-only body is enough to turn evaluators
// on. Returns true if this call created the namespace, so the teardown can
// delete it only when safe.
func ensureIstioCR(t *testing.T) bool {
	t.Helper()
	// `kubectl get ns` first so we know whether to clean it up later.
	// Checking existence via kubectl avoids an extra client dependency and
	// matches the rest of the suite's shell-out style.
	createdNS := false
	if _, err := utils.Run(exec.Command("kubectl", "get", "ns", istioCRNamespace)); err != nil {
		if _, err := utils.Run(exec.Command("kubectl", "create", "ns", istioCRNamespace)); err != nil {
			t.Fatalf("create namespace %s: %v", istioCRNamespace, err)
		}
		createdNS = true
	}

	manifest := fmt.Sprintf(`
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: %s
  namespace: %s
`, istioCRName, istioCRNamespace)
	applyManifest(t, manifest)

	// Give the controller's cached Snapshot a chance to pick up the new CR
	// before the next subtest races to assert on events.
	time.Sleep(5 * time.Second)
	return createdNS
}

func createWorkloadNamespace(t *testing.T, suffix string) string {
	t.Helper()
	ns := fmt.Sprintf("restarter-e2e-%s", suffix)
	if _, err := utils.Run(exec.Command("kubectl", "create", "ns", ns)); err != nil {
		t.Fatalf("create namespace %s: %v", ns, err)
	}
	t.Cleanup(func() {
		_, _ = utils.Run(exec.Command("kubectl", "delete", "ns", ns, "--ignore-not-found"))
	})
	return ns
}

// applyManifest pipes a YAML manifest through kubectl apply via a temp file.
// A temp file avoids escaping the manifest body as a shell arg.
func applyManifest(t *testing.T, manifest string) {
	t.Helper()
	f, err := os.CreateTemp("", "manifest-*.yaml")
	if err != nil {
		t.Fatalf("create temp manifest: %v", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(manifest); err != nil {
		t.Fatalf("write temp manifest: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp manifest: %v", err)
	}
	absPath, err := filepath.Abs(f.Name())
	if err != nil {
		t.Fatalf("resolve temp manifest path: %v", err)
	}
	if _, err := utils.Run(exec.Command("kubectl", "apply", "-f", absPath)); err != nil {
		t.Fatalf("kubectl apply: %v", err)
	}
}

// waitForOnePod polls until at least one pod matching selector exists in ns.
// Pods do not need to reach Running — the restarter inspects spec, so Pending
// (ImagePullBackOff on the outdated sidecar image) is enough to trigger
// evaluation.
func waitForOnePod(t *testing.T, ns, selector string) {
	t.Helper()
	pollUntil(t, 1*time.Minute, defaultInterval, func() error {
		out, err := utils.Run(exec.Command("kubectl", "get", "pods", "-n", ns,
			"-l", selector, "-o", "jsonpath={.items[*].metadata.name}"))
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) == "" {
			return fmt.Errorf("no pods matching %s in %s yet", selector, ns)
		}
		return nil
	})
}

// waitForEventReason polls for a Kubernetes Event with the given reason
// scoped to the specified involvedObject.
func waitForEventReason(t *testing.T, ns, kind, name, reason string) {
	t.Helper()
	t.Logf("waiting for %s event on %s/%s in %s", reason, kind, name, ns)
	pollUntil(t, defaultTimeout, defaultInterval, func() error {
		out, err := utils.Run(exec.Command("kubectl", "get", "events", "-n", ns,
			"--field-selector",
			fmt.Sprintf("involvedObject.kind=%s,involvedObject.name=%s,reason=%s", kind, name, reason),
			"-o", "jsonpath={.items[*].reason}"))
		if err != nil {
			return err
		}
		if !strings.Contains(out, reason) {
			return fmt.Errorf("event %s not yet observed", reason)
		}
		return nil
	})
}

// assertNoEventWithReason fails the test if any event with reason exists on
// the specified involvedObject in ns.
func assertNoEventWithReason(t *testing.T, ns, kind, name, reason string) {
	t.Helper()
	out, err := utils.Run(exec.Command("kubectl", "get", "events", "-n", ns,
		"--field-selector",
		fmt.Sprintf("involvedObject.kind=%s,involvedObject.name=%s,reason=%s", kind, name, reason),
		"-o", "jsonpath={.items[*].reason}"))
	if err != nil {
		t.Fatalf("list events for %s/%s in %s: %v", kind, name, ns, err)
	}
	if strings.Contains(out, reason) {
		t.Fatalf("unexpected %s event on %s/%s in %s: %s", reason, kind, name, ns, out)
	}
}

// deploymentWithSidecar renders a Deployment manifest with a pre-baked
// istio-proxy container. In production the istio-proxy container is injected
// by the Istio mutating webhook; here we hand-craft it so the test cluster
// does not need istiod. The sidecar.istio.io/inject=false annotation is a
// defense-in-depth guard in case the test runs on a cluster that happens to
// have Istio installed — it prevents the webhook from re-mutating our pods.
func deploymentWithSidecar(name, ns, sidecarImage string) string {
	return fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
      annotations:
        sidecar.istio.io/inject: "false"
    spec:
      containers:
        - name: app
          image: %s
        - name: istio-proxy
          image: %s
`, name, ns, name, name, name, appContainerImage, sidecarImage)
}

// statefulSetWithSidecar renders a StatefulSet manifest with a pre-baked
// istio-proxy container; see deploymentWithSidecar for the rationale. The
// serviceName field is required by the StatefulSet schema but no matching
// Service is needed for restarter evaluation.
func statefulSetWithSidecar(name, ns, sidecarImage string) string {
	return fmt.Sprintf(`
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: %s
  namespace: %s
spec:
  serviceName: %s
  replicas: 1
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
      annotations:
        sidecar.istio.io/inject: "false"
    spec:
      containers:
        - name: app
          image: %s
        - name: istio-proxy
          image: %s
`, name, ns, name, name, name, appContainerImage, sidecarImage)
}

func daemonSetWithSidecar(name, ns, sidecarImage string) string {
	return fmt.Sprintf(`
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: %s
  namespace: %s
spec:
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
      annotations:
        sidecar.istio.io/inject: "false"
    spec:
      containers:
        - name: app
          image: %s
        - name: istio-proxy
          image: %s
`, name, ns, name, name, appContainerImage, sidecarImage)
}

// replicaSetWithSidecar renders a bare ReplicaSet (no OwnerReferences) with a
// pre-baked istio-proxy container; see deploymentWithSidecar for the rationale.
func replicaSetWithSidecar(name, ns, sidecarImage string) string {
	return fmt.Sprintf(`
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: %s
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
      annotations:
        sidecar.istio.io/inject: "false"
    spec:
      containers:
        - name: app
          image: %s
        - name: istio-proxy
          image: %s
`, name, ns, name, name, appContainerImage, sidecarImage)
}

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken(t *testing.T) (string, error) {
	t.Helper()
	const tokenRequestRawString = `{
				"apiVersion": "authentication.k8s.io/v1",
				"kind": "TokenRequest"
			}`

	f, err := os.CreateTemp("", "token-request-*.json")
	if err != nil {
		return "", fmt.Errorf("create token request temp file: %w", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(tokenRequestRawString); err != nil {
		return "", fmt.Errorf("write token request temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close token request temp file: %w", err)
	}
	tokenRequestFile, err := filepath.Abs(f.Name())
	if err != nil {
		return "", fmt.Errorf("resolve token request path: %w", err)
	}

	var out string
	pollUntil(t, 2*time.Minute, time.Second, func() error {
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("kubectl create token: %w (output: %s)", err, string(output))
		}

		var token tokenRequest
		if err := json.Unmarshal(output, &token); err != nil {
			return fmt.Errorf("parse token response: %w", err)
		}
		out = token.Status.Token
		return nil
	})

	return out, nil
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() (string, error) {
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
