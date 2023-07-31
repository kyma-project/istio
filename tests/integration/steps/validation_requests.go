package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/kyma-project/istio/operator/tests/integration/testsupport"
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"strings"
)

func EveryInstanceInNamespaceHasEnvoyReferer(ctx context.Context, name string, namespace string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}
	var podList corev1.PodList
	err = k8sClient.List(ctx, &podList, &client.ListOptions{Namespace: namespace})
	pods := funk.Filter(podList.Items, func(pod corev1.Pod) bool { return strings.HasPrefix(pod.Name, name) }).([]corev1.Pod)

	cfg, err := config.GetConfig()
	if err != nil {
		return ctx, err
	}

	tripper, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return ctx, err
	}
	for _, item := range pods {
		path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", item.Namespace, item.Name)
		hostIP := strings.TrimLeft(cfg.Host, "htps:/")
		serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}

		dialer := spdy.NewDialer(upgrader, &http.Client{Transport: tripper}, http.MethodPost, &serverURL)
		stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
		out, errOut := new(bytes.Buffer), new(bytes.Buffer)

		forwarder, err := portforward.New(dialer, []string{"15000:15000"}, stopChan, readyChan, out, errOut)
		if err != nil {
			return ctx, err
		}

		go func() {
			if err = forwarder.ForwardPorts(); err != nil { // Locks until stopChan is closed.
				return
			}
		}()

		select {
		case <-readyChan:
			break
		}

		response, err := http.Get("http://localhost:15000/config_dump?mask=bootstrap")
		if err != nil {
			return ctx, err
		}

		forwarder.Close()
		raw, err := io.ReadAll(response.Body)
		if err != nil {
			return ctx, err
		}

		type layer struct {
			Name        string            `json:"name"`
			StaticLayer map[string]string `json:"static_layer"`
		}

		var configs struct {
			Configs []struct {
				Bootstrap struct {
					LayeredRuntime struct {
						Layers []layer `json:"layers"`
					} `json:"layered_runtime"`
				} `json:"bootstrap"`
			} `json:"configs"`
		}
		err = json.Unmarshal(raw, &configs)
		if err != nil {
			return ctx, err
		}

		l := funk.Filter(configs.Configs[0].Bootstrap.LayeredRuntime.Layers, func(obj layer) bool {
			return obj.Name == "kyma_referer"
		}).([]layer)

		if len(l) != 1 {
			return ctx, fmt.Errorf("pod %s doesn't have referer envoy filter configured", item.Name)
		}

		if val, ok := l[0].StaticLayer["envoy.reloadable_features.http_allow_partial_urls_in_referer"]; ok {
			if val != "false" {
				return ctx, errors.New("Value of envoy.reloadable_features.http_allow_partial_urls_in_referer isn't \"false\"")
			}
		} else {
			return ctx, errors.New("Value of envoy.reloadable_features.http_allow_partial_urls_in_referer not present")
		}
	}
	return ctx, nil
}

// ValidateHeader validates that the header givenHeaderName with value givenHeaderValue is forwarded to the application as header expectedHeaderName with the value expectedHeaderValue.
func ValidateHeader(ctx context.Context, givenHeaderName, givenHeaderValue, expectedHeaderName, expectedHeaderValue string) (context.Context, error) {

	ingressAddress, err := fetchIstioIngressGatewayAddress(ctx)
	if err != nil {
		return ctx, err
	}

	c := testsupport.NewHttpClientWithRetry()
	headers := map[string]string{
		givenHeaderName: givenHeaderValue,
	}
	url := fmt.Sprintf("http://%s/get?show_env=true", ingressAddress)
	asserter := testsupport.BodyContainsAsserter{
		Expected: []string{
			fmt.Sprintf(`"%s": "%s"`, expectedHeaderName, expectedHeaderValue),
		},
	}

	return ctx, c.GetWithHeaders(url, headers, asserter)
}

func fetchIstioIngressGatewayAddress(ctx context.Context) (string, error) {

	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return "", err
	}

	istioIngressGatewayNamespaceName := types.NamespacedName{
		Name:      "istio-ingressgateway",
		Namespace: "istio-system",
	}

	var ingressIp string
	var ingressPort int32

	err = retry.Do(func() error {

		runsOnGardener, err := testsupport.RunsOnGardener(ctx, k8sClient)
		if err != nil {
			return err
		}

		if runsOnGardener {
			svc := corev1.Service{}
			if err := k8sClient.Get(context.TODO(), istioIngressGatewayNamespaceName, &svc); err != nil {
				return err
			}

			if len(svc.Status.LoadBalancer.Ingress) == 0 {
				return errors.New("no ingress ip found")
			} else {
				ingressIp = svc.Status.LoadBalancer.Ingress[0].IP
				for _, port := range svc.Spec.Ports {
					if port.Name == "http2" {
						ingressPort = port.Port
					}
				}
				return nil
			}
		} else {
			// In case we are not running on Gardener we assume that it's a k3d cluster that has 127.0.0.1 as default address
			ingressIp = "localhost"
			ingressPort = 80
		}

		return nil
	}, testcontext.GetRetryOpts()...)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%d", ingressIp, ingressPort), nil
}
