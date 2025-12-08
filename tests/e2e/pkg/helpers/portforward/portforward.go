package portforward

import (
	"context"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/test_writer"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"testing"
	"time"
)

func CreateControllerPortForwading(t *testing.T) (host string, port int, err error) {
	config := client.GetKubeConfig(t)

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return "", 0, err
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", 0, err
	}

	pods, err := k8sClient.CoreV1().Pods("kyma-system").List(t.Context(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=istio-operator",
	})
	if err != nil {
		return "", 0, err
	}

	url := k8sClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace("kyma-system").
		Name(pods.Items[0].Name).
		SubResource("portforward").URL()

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, url)

	ports := []string{"8080:8080"}

	fw, err := portforward.New(dialer, ports, stopChan, readyChan, test_writer.NewTLogWriter(t), test_writer.NewTLogWriter(t))
	if err != nil {
		return "", 0, err
	}

	setup.DeclareCleanup(t, func() {
		t.Log("Cleaning up Istio operator port-forwarding...")
		close(stopChan)
		fw.Close()
		t.Log("Port-forward for Istio operator stopped")
	})

	t.Logf("Creating port-forward for Istio operator...; url: %s, ports: %v", url.String(), ports)
	go func() {
		err = fw.ForwardPorts()
		if err != nil {
			t.Logf("Failed to forward ports: %v", err)
			return
		}
	}()
	<-fw.Ready
	return "localhost", 8080, nil
}

func CreateIngressGatewayPortForwarding(t *testing.T) (host string, port int, err error) {
	config := client.GetKubeConfig(t)

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return "", 0, err
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", 0, err
	}

	pods, err := k8sClient.CoreV1().Pods("istio-system").List(t.Context(), metav1.ListOptions{
		LabelSelector: "app=istio-ingressgateway",
	})
	if err != nil {
		return "", 0, err
	}

	// wait until all pods are running
	err = wait.For(func(ctx context.Context) (done bool, err error) {
		pods, err = k8sClient.CoreV1().Pods("istio-system").List(t.Context(), metav1.ListOptions{
			LabelSelector: "app=istio-ingressgateway",
		})
		if err != nil {
			return false, err
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase != "Running" {
				return false, nil
			}
		}
		return true, nil
	}, wait.WithTimeout(15*time.Second), wait.WithInterval(1*time.Second))

	url := k8sClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace("istio-system").
		Name(pods.Items[0].Name).
		SubResource("portforward").URL()

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, url)

	ports := []string{"8080:8080"}

	fw, err := portforward.New(dialer, ports, stopChan, readyChan, test_writer.NewTLogWriter(t), test_writer.NewTLogWriter(t))
	if err != nil {
		return "", 0, err
	}

	setup.DeclareCleanup(t, func() {
		t.Log("Cleaning up Istio ingress-gateway port-forwarding...")
		close(stopChan)
		fw.Close()
		t.Log("Port-forward for Istio ingress-gateway stopped")
	})

	t.Logf("Creating port-forward for Istio ingress-gateway...; url: %s, ports: %v", url.String(), ports)
	go func() {
		err = fw.ForwardPorts()
		if err != nil {
			t.Logf("Failed to forward ports: %v", err)
			return
		}
	}()
	<-fw.Ready
	return "local.kyma.dev", 8080, nil
}
