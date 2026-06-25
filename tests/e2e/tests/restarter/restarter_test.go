package restarter

import (
	"context"
	"embed"
	"sync"
	"testing"
	"time"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

//go:embed testdata
var testdata embed.FS

// testing ingressgateway restarts
func TestRestarter_IngressGateway(t *testing.T) {
	t.Run("NumTrustedProxies changed, gateway restarted", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		dep := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "istio-ingressgateway", Namespace: "istio-system"}}
		require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(dep.GetName(), dep.GetNamespace())))
		require.NoError(t, c.Get(t.Context(), dep.GetName(), dep.GetNamespace(), &dep))
		podUIDs := GetPodUIDs(t, c, "app=istio-ingressgateway", "istio-system")

		// Update IstioCR in kyma-system with numTrustedProxies=1
		err = modulehelpers.NewIstioCRBuilder().
			WithNumTrustedProxies(1).
			Update(t)
		require.NoError(t, err)

		AssertDeploymentRestarted(t, c, &dep)
		AssertPodsRecreated(t, c, podUIDs, "app=istio-ingressgateway", "istio-system")
	})
	t.Run("TrustDomain changed, gateway restarted", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		dep := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "istio-ingressgateway", Namespace: "istio-system"}}
		require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(dep.GetName(), dep.GetNamespace())))
		require.NoError(t, c.Get(t.Context(), dep.GetName(), dep.GetNamespace(), &dep))
		podUIDs := GetPodUIDs(t, c, "app=istio-ingressgateway", "istio-system")

		// Update IstioCR in kyma-system with numTrustedProxies=1
		err = modulehelpers.NewIstioCRBuilder().
			WithTrustDomain("trust.com").
			Update(t)
		require.NoError(t, err)

		AssertDeploymentRestarted(t, c, &dep)
		AssertPodsRecreated(t, c, podUIDs, "app=istio-ingressgateway", "istio-system")
	})
	t.Run("ForwardClientCertDetails changed, gateway restarted", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		dep := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "istio-ingressgateway", Namespace: "istio-system"}}
		require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(dep.GetName(), dep.GetNamespace())))
		require.NoError(t, c.Get(t.Context(), dep.GetName(), dep.GetNamespace(), &dep))
		podUIDs := GetPodUIDs(t, c, "app=istio-ingressgateway", "istio-system")

		// Update IstioCR in kyma-system with numTrustedProxies=1
		err = modulehelpers.NewIstioCRBuilder().
			WithForwardClientCertDetails(operatorv1alpha2.Sanitize).
			Update(t)
		require.NoError(t, err)

		AssertDeploymentRestarted(t, c, &dep)
		AssertPodsRecreated(t, c, podUIDs, "app=istio-ingressgateway", "istio-system")
	})
}

func TestRestarter_Workload(t *testing.T) {
	t.Run("Sidecar images differ on 250 pods, all workloads restarted", func(t *testing.T) {
		testNamespace := "test-workload-restart"
		invalidImage := "europe-docker.pkg.dev/kyma-project/prod/external/istio/proxyv2:1.21.3-distroless"
		numDepl := 100
		numDeplToRestart := 50

		c, err := client.ResourcesClient(t)
		require.NoError(t, err)
		require.NoError(t, infrastructure.CreateNamespace(t, testNamespace, infrastructure.WithSidecarInjectionEnabled()))
		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)

		// concurrently create test deployments
		wg := sync.WaitGroup{}
		CreateDeploymentsWaitGroup(t, c, &wg, "testdata/deployment.yaml", testNamespace, numDepl)
		CreateDeploymentsWaitGroup(t, c, &wg, "testdata/deployment_to_restart.yaml", testNamespace, numDeplToRestart)
		wg.Wait()

		// patch pods with invalid image
		podList := &corev1.PodList{}
		require.NoError(t, c.List(t.Context(), podList, resources.WithLabelSelector("app=fake-to-restart"),
			resources.WithFieldSelector("metadata.namespace="+testNamespace)))
		wg = sync.WaitGroup{}
		for _, pod := range podList.Items {
			wg.Go(func() {
				patchErr := c.Patch(t.Context(), &pod, k8s.Patch{
					PatchType: types.StrategicMergePatchType,
					Data:      []byte(`{"spec":{"initContainers":[{"name":"istio-proxy","image":"` + invalidImage + `"}]}}`),
				})
				require.NoError(t, patchErr)
				require.NoError(t, wait.For(conditions.New(c).PodReady(&pod)))
			})
		}
		wg.Wait()
		// assert restarts occurred on modified pods and deployments
		oldPodUIDs := GetPodUIDs(t, c, "app=fake-to-restart", testNamespace)

		restartedDeployments := &appsv1.DeploymentList{}
		require.NoError(t, c.List(t.Context(), restartedDeployments, resources.WithLabelSelector("app=fake-to-restart"),
			resources.WithFieldSelector("metadata.namespace="+testNamespace)))

		notRestartedDeployments := appsv1.DeploymentList{}
		require.NoError(t, c.List(t.Context(), &notRestartedDeployments, resources.WithLabelSelector("app=fake-deployment"),
			resources.WithFieldSelector("metadata.namespace="+testNamespace)))

		// we need to update istioCR with whatever to trigger restarts due to reconciler constraints
		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithNumTrustedProxies(1).Update(t))

		// concurrently wait for deployment restart
		wg = sync.WaitGroup{}
		for _, depl := range restartedDeployments.Items {
			wg.Go(func() {
				AssertDeploymentRestarted(t, c, &depl)
			})
		}
		// concurrently check if deployments did not restart
		for _, depl := range notRestartedDeployments.Items {
			wg.Go(func() {
				// deployment here must be Available
				require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(depl.GetName(), depl.GetNamespace())))
				AssertDeploymentDidNotRestart(t, c, &depl)
			})
		}
		wg.Wait()

		err = wait.For(func(ctx context.Context) (bool, error) {
			newPods := &corev1.PodList{}
			err = c.List(ctx, newPods,
				resources.WithLabelSelector("app=fake-to-restart"),
				resources.WithFieldSelector("metadata.namespace="+testNamespace))
			if err != nil {
				return false, err
			}
			for _, pod := range newPods.Items {
				if s := GetPodImage(pod, "istio-proxy"); s == invalidImage {
					return false, nil
				}
				for _, oldUID := range oldPodUIDs {
					if pod.GetUID() == oldUID {
						return false, nil
					}
				}
			}
			return true, nil

		}, wait.WithTimeout(time.Minute*5))
		assert.NoError(t, err)
	})
	t.Run("DNSProxying changed, workload restarted", func(t *testing.T) {
		testNamespace := "test-dnsproxying-restart"

		c, err := client.ResourcesClient(t)
		require.NoError(t, err)
		require.NoError(t, infrastructure.CreateNamespace(t, testNamespace, infrastructure.WithSidecarInjectionEnabled()))
		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		depl := appsv1.Deployment{}
		require.NoError(t, decoder.DecodeFile(testdata, "testdata/deployment_to_restart.yaml", &depl))
		depl.SetNamespace(testNamespace)
		require.NoError(t, c.Create(t.Context(), &depl))
		require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(depl.GetName(), depl.GetNamespace())))

		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithEnableDNSProxying(true).Update(t))
		AssertDeploymentRestarted(t, c, &depl)
	})
	t.Run("Istio sidecar resources changed, workload restarted", func(t *testing.T) {
		testNamespace := "test-istio-sidecar-restart"
		wantedProxyCPU := "4"
		wantedProxyMemory := "4Gi"
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)
		require.NoError(t, infrastructure.CreateNamespace(t, testNamespace, infrastructure.WithSidecarInjectionEnabled()))
		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		depl := appsv1.Deployment{}
		require.NoError(t, decoder.DecodeFile(testdata, "testdata/deployment_to_restart.yaml", &depl))
		depl.SetNamespace(testNamespace)
		require.NoError(t, c.Create(t.Context(), &depl))
		require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(depl.GetName(), depl.GetNamespace())))
		require.NoError(t, c.Patch(t.Context(), &depl, k8s.Patch{
			PatchType: types.StrategicMergePatchType,
			Data:      []byte(`{"spec":{"template":{"metadata":{"annotations":{"sidecar.istio.io/proxyCPU": "` + wantedProxyCPU + `", "sidecar.istio.io/proxyMemory": "` + wantedProxyMemory + `"}}}}}`),
		}))
		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithEnableDNSProxying(true).Update(t))
		AssertDeploymentRestarted(t, c, &depl)
		assert.NoError(t, wait.For(func(ctx context.Context) (bool, error) {
			newPods := &corev1.PodList{}
			err = c.List(ctx, newPods, resources.WithLabelSelector("app=fake-to-restart"),
				resources.WithFieldSelector("metadata.namespace="+testNamespace))
			if err != nil {
				return false, err
			}
			for _, pod := range newPods.Items {
				for _, container := range pod.Spec.InitContainers {
					if container.Name != "istio-proxy" {
						continue
					}
					observedCPU := container.Resources.Requests.Cpu().String()
					observedMemory := container.Resources.Requests.Memory().String()
					if observedCPU != wantedProxyCPU || observedMemory != wantedProxyMemory {
						return false, nil
					}
					break
				}
			}
			return true, nil
		}, wait.WithTimeout(time.Minute*5)))
	})
	t.Run("PrometheusMerge changed, workload restarted", func(t *testing.T) {
		testNamespace := "test-prometheus-restart"
		defaultPrometheusPort := "15020"
		defaultPrometheusPath := "/stats/prometheus"
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)
		require.NoError(t, infrastructure.CreateNamespace(t, testNamespace, infrastructure.WithSidecarInjectionEnabled()))
		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)

		depl := appsv1.Deployment{}
		require.NoError(t, decoder.DecodeFile(testdata, "testdata/deployment_to_restart.yaml", &depl))
		depl.SetNamespace(testNamespace)
		require.NoError(t, c.Create(t.Context(), &depl))
		require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(depl.GetName(), depl.GetNamespace())))
		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithPrometheusMerge(true).Update(t))

		AssertDeploymentRestarted(t, c, &depl)
		assert.NoError(t, wait.For(func(ctx context.Context) (bool, error) {
			newPods := &corev1.PodList{}
			err = c.List(ctx, newPods, resources.WithLabelSelector("app=fake-to-restart"),
				resources.WithFieldSelector("metadata.namespace="+testNamespace))
			if err != nil {
				return false, err
			}
			for _, pod := range newPods.Items {
				annotations := pod.GetAnnotations()
				if annotations != nil {
					observedPrometheusPort := annotations["prometheus.io/port"]
					observedPrometheusPath := annotations["prometheus.io/path"]
					if observedPrometheusPort != defaultPrometheusPort || observedPrometheusPath != defaultPrometheusPath {
						return false, nil
					}
				}
			}
			return true, nil
		}))
	})
	t.Run("Workload is restarted only once", func(t *testing.T) {
		testNamespace := "test-workload-restart-once"
		invalidImage := "europe-docker.pkg.dev/kyma-project/prod/external/istio/proxyv2:1.21.3-distroless"

		c, err := client.ResourcesClient(t)
		require.NoError(t, err)
		require.NoError(t, infrastructure.CreateNamespace(t, testNamespace, infrastructure.WithSidecarInjectionEnabled()))
		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)

		depl := appsv1.Deployment{}
		require.NoError(t, decoder.DecodeFile(testdata, "testdata/deployment_to_restart.yaml", &depl))
		depl.SetNamespace(testNamespace)
		require.NoError(t, c.Create(t.Context(), &depl))
		require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(depl.GetName(), depl.GetNamespace())))

		pods := corev1.PodList{}
		require.NoError(t, c.List(context.Background(), &pods, resources.WithLabelSelector("app=fake-to-restart"),
			resources.WithFieldSelector("metadata.namespace="+testNamespace)))
		wg := sync.WaitGroup{}
		for _, pod := range pods.Items {
			wg.Go(func() {
				require.NoError(t, c.Patch(t.Context(), &pod, k8s.Patch{
					PatchType: types.StrategicMergePatchType,
					Data:      []byte(`{"spec":{"initContainers":[{"name":"istio-proxy","image":"` + invalidImage + `"}]}}`),
				}))
				require.NoError(t, wait.For(conditions.New(c).PodReady(&pod)))
			})
		}
		wg.Wait()
		// trigger reconciliation
		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithNumTrustedProxies(1).Update(t))
		AssertDeploymentRestarted(t, c, &depl)
		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithNumTrustedProxies(2).Update(t))
		AssertDeploymentDidNotRestart(t, c, &depl)
	})
	t.Run("Deployment with failed pods is not restarted", func(t *testing.T) {
		testNamespace := "test-failed-pods-restart"
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)
		require.NoError(t, infrastructure.CreateNamespace(t, testNamespace, infrastructure.WithSidecarInjectionEnabled()))
		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		depl := appsv1.Deployment{}
		require.NoError(t, decoder.DecodeFile(testdata, "testdata/failed_deployment.yaml", &depl))
		depl.SetNamespace(testNamespace)
		require.NoError(t, c.Create(t.Context(), &depl))
		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithEnableDNSProxying(true).Update(t))
		AssertDeploymentDidNotRestart(t, c, &depl)
	})
	t.Run("StatefulSet is restarted", func(t *testing.T) {
		testNamespace := "test-statefulset-restart"
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)
		require.NoError(t, infrastructure.CreateNamespace(t, testNamespace, infrastructure.WithSidecarInjectionEnabled()))
		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		ss := appsv1.StatefulSet{}
		require.NoError(t, decoder.DecodeFile(testdata, "testdata/statefulset_to_restart.yaml", &ss))
		ss.SetNamespace(testNamespace)
		require.NoError(t, c.Create(t.Context(), &ss))
		require.NoError(t, wait.For(conditions.New(c).ResourceMatch(&ss, func(obj k8s.Object) bool {
			observed := obj.(*appsv1.StatefulSet).Status
			return observed.ReadyReplicas == observed.Replicas && observed.AvailableReplicas == observed.Replicas
		})))
		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithEnableDNSProxying(true).Update(t))

		oldGeneration := ss.GetGeneration()
		oldRestartedAt := ss.Spec.Template.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"]
		assert.NoError(t, wait.For(conditions.New(c).ResourceMatch(&ss, func(obj k8s.Object) bool {
			observed := obj.(*appsv1.StatefulSet).Status
			return observed.ReadyReplicas == observed.Replicas && observed.AvailableReplicas == observed.Replicas
		})))
		assert.NoError(t, wait.For(conditions.New(c).ResourceMatch(&ss, func(obj k8s.Object) bool {
			observed := obj.(*appsv1.StatefulSet)
			return observed.GetGeneration() != oldGeneration &&
				observed.Spec.Template.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"] != oldRestartedAt
		})))
	})
	t.Run("DaemonSet is restarted", func(t *testing.T) {
		testNamespace := "test-daemonset-restart"
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)
		require.NoError(t, infrastructure.CreateNamespace(t, testNamespace, infrastructure.WithSidecarInjectionEnabled()))
		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		ds := appsv1.DaemonSet{}
		require.NoError(t, decoder.DecodeFile(testdata, "testdata/daemonset_to_restart.yaml", &ds))
		ds.SetNamespace(testNamespace)
		require.NoError(t, c.Create(t.Context(), &ds))
		require.NoError(t, wait.For(conditions.New(c).DaemonSetReady(&ds)))
		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithEnableDNSProxying(true).Update(t))
		oldGeneration := ds.GetGeneration()
		oldRestartedAt := ds.Spec.Template.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"]
		assert.NoError(t, wait.For(conditions.New(c).DaemonSetReady(&ds)))
		assert.NoError(t, wait.For(conditions.New(c).ResourceMatch(&ds, func(obj k8s.Object) bool {
			observed := obj.(*appsv1.DaemonSet)
			return observed.GetGeneration() != oldGeneration &&
				observed.Spec.Template.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"] != oldRestartedAt
		}), wait.WithTimeout(time.Minute*5)))
	})
	t.Run("Bare pod is not restarted, istio in Warning state", func(t *testing.T) {
		// Istio module does not support restarting Pods without OwnerReferences
		testNamespace := "test-bare-pod-restart"
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)
		require.NoError(t, infrastructure.CreateNamespace(t, testNamespace, infrastructure.WithSidecarInjectionEnabled()))
		istioCR, err := modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		pod := corev1.Pod{}
		require.NoError(t, decoder.DecodeFile(testdata, "testdata/pod_to_restart.yaml", &pod))
		pod.SetNamespace(testNamespace)
		require.NoError(t, c.Create(t.Context(), &pod))
		require.NoError(t, wait.For(conditions.New(c).PodReady(&pod)))
		istioCR.Spec.Config.EnableDNSProxying = new(true)
		require.NoError(t, c.Update(t.Context(), istioCR))
		oldUID := pod.GetUID()
		oldGeneration := pod.GetGeneration()
		assert.NoError(t, wait.For(conditions.New(c).ResourceMatch(&pod, func(obj k8s.Object) bool {
			observed := obj.(*corev1.Pod)
			return observed.UID == oldUID && observed.GetGeneration() == oldGeneration
		})))
		assert.NoError(t, wait.For(conditions.New(c).ResourceMatch(istioCR, func(obj k8s.Object) bool {
			observed := obj.(*operatorv1alpha2.Istio).Status
			return observed.State == operatorv1alpha2.Warning
		})))
	})
	t.Run("Pod from running Job is not restarted, Istio in Warning state", func(t *testing.T) {
		// Istio module does not support restarting running Pods owned by Jobs
		testNamespace := "test-job-restart"
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)
		require.NoError(t, infrastructure.CreateNamespace(t, testNamespace, infrastructure.WithSidecarInjectionEnabled()))
		istioCR, err := modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		job := batchv1.Job{}
		require.NoError(t, decoder.DecodeFile(testdata, "testdata/job_to_restart.yaml", &job))
		job.SetNamespace(testNamespace)
		require.NoError(t, c.Create(t.Context(), &job))

		istioCR.Spec.Config.EnableDNSProxying = new(true)
		require.NoError(t, c.Update(t.Context(), istioCR))

		oldGeneration := job.GetGeneration()
		assert.NoError(t, wait.For(conditions.New(c).ResourceMatch(&job, func(obj k8s.Object) bool {
			observed := obj.(*batchv1.Job)
			return observed.GetGeneration() == oldGeneration
		})))
		assert.NoError(t, wait.For(conditions.New(c).ResourceMatch(istioCR, func(obj k8s.Object) bool {
			observed := obj.(*operatorv1alpha2.Istio).Status
			return observed.State == operatorv1alpha2.Warning
		})))
	})
}

func CreateDeploymentsWaitGroup(t *testing.T, r *resources.Resources, wg *sync.WaitGroup, fileName, namespace string, numDeployments int) {
	t.Helper()
	deplTmpl := &appsv1.Deployment{}
	require.NoError(t, decoder.DecodeFile(testdata, fileName, deplTmpl))
	for i := 0; i < numDeployments; i++ {
		wg.Go(func() {
			depl := deplTmpl.DeepCopy()
			depl.SetNamespace(namespace)
			require.NoError(t, r.Create(t.Context(), depl))
			require.NoError(t, wait.For(conditions.New(r).DeploymentAvailable(depl.GetName(), depl.GetNamespace())))
		})
	}
}

func GetPodImage(pod corev1.Pod, containerName string) string {
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			return container.Image
		}
	}
	return ""
}

func GetPodUIDs(t *testing.T, r *resources.Resources, labels, namespace string) []types.UID {
	t.Helper()
	var podUIDs []types.UID
	podList := &corev1.PodList{}
	require.NoError(t, r.List(t.Context(), podList, resources.WithLabelSelector(labels),
		resources.WithFieldSelector("metadata.namespace="+namespace)))
	for _, pod := range podList.Items {
		podUIDs = append(podUIDs, pod.GetUID())
	}
	return podUIDs
}

func AssertDeploymentRestarted(t *testing.T, r *resources.Resources, depl *appsv1.Deployment) {
	t.Helper()
	oldGeneration := depl.GetGeneration()
	oldRestartedAt := depl.Spec.Template.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"]
	assert.NoError(t, wait.For(conditions.New(r).DeploymentAvailable(depl.GetName(), depl.GetNamespace())))
	assert.NoError(t, wait.For(conditions.New(r).ResourceMatch(depl, func(obj k8s.Object) bool {
		observed := obj.(*appsv1.Deployment)
		return observed.GetGeneration() != oldGeneration &&
			observed.Spec.Template.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"] != oldRestartedAt
	}), wait.WithTimeout(time.Minute*2)))
}

func AssertDeploymentDidNotRestart(t *testing.T, r *resources.Resources, depl *appsv1.Deployment) {
	t.Helper()
	oldGeneration := depl.GetGeneration()
	oldRestartedAt := depl.Spec.Template.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"]
	assert.NoError(t, wait.For(conditions.New(r).ResourceMatch(depl, func(obj k8s.Object) bool {
		observed := obj.(*appsv1.Deployment)
		return observed.GetGeneration() == oldGeneration &&
			observed.Spec.Template.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"] == oldRestartedAt
	}), wait.WithTimeout(time.Minute*2)))
}

func AssertPodsRecreated(t *testing.T, r *resources.Resources, oldUIDs []types.UID, labelSelector, namespace string) {
	t.Helper()
	assert.NoError(t, wait.For(func(ctx context.Context) (bool, error) {
		newPods := &corev1.PodList{}
		if err := r.List(ctx, newPods,
			resources.WithLabelSelector(labelSelector),
			resources.WithFieldSelector("metadata.namespace="+namespace),
		); err != nil {
			return false, err
		}
		for _, pod := range newPods.Items {
			for _, oldUID := range oldUIDs {
				if pod.GetUID() == oldUID {
					return false, nil
				}
			}
		}
		return true, nil
	}, wait.WithTimeout(time.Minute*5)))
}
