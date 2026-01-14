package zero_downtime

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"

	"github.com/cucumber/godog"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/tests/integration/pkg/ip"

	"log"
	"net/http"
	"strings"
	"time"
)

const requestTimeout = 10 * time.Second
const testInterval = 500 * time.Millisecond
const numberOfThreads = 5

// ZeroDowntimeTestRunner holds Tester instances created by the StartZeroDowntimeTest function,
// so then the FinishZeroDowntimeTests function knows which Testers to stop
// Every test scenario instance should have own copy of this structure to allow parallel execution of tests
type ZeroDowntimeTestRunner struct {
	testers []Tester
}

func getClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: requestTimeout,
	}
}

func doSingleRequest(client *http.Client, host string, path string, lbUrl string) error {
	req, err := http.NewRequest(http.MethodGet, lbUrl, nil)
	if err != nil {
		return err
	}
	req.Host = host

	now := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request done at %s to host %s path %s failed because of: %w", now, host, path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request done at %s to host %s path %s failed with status code: %d", now, host, path, resp.StatusCode)
	}

	time.Sleep(testInterval)
	return nil
}

func (zd *ZeroDowntimeTestRunner) StartZeroDowntimeTest(ctx context.Context, c client.Client, host string, path string) (context.Context, error) {
	ingressAddress, err := fetchIstioIngressGatewayAddress(ctx, c)
	if err != nil {
		return ctx, err
	}
	lbUrl := fmt.Sprintf("http://%s/%s", ingressAddress, strings.TrimLeft(path, "/"))
	client := getClient()

	testFn := func() error {
		return doSingleRequest(client, host, path, lbUrl)
	}

	tester := NewTester(fmt.Sprintf("host-%s", host), testFn, numberOfThreads)
	zd.testers = append(zd.testers, tester)

	log.Printf("Starting zero downtime tester %s", tester.Name())
	tester.Start()

	return ctx, nil
}

func (zd *ZeroDowntimeTestRunner) FinishZeroDowntimeTests(ctx context.Context) (context.Context, error) {
	allErrs := make([]error, 0)

	for _, tester := range zd.testers {
		log.Printf("Stopping zero downtime tester %s", tester.Name())
		results := tester.Stop()
		for _, result := range results {
			if result.Err != nil {
				log.Printf("Got result from worker %s: requests: %d, error: %v", result.WorkerName, result.TestCount, result.Err)
				allErrs = append(allErrs, result.Err)
			} else {
				log.Printf("Got result from worker %s: requests: %d", result.WorkerName, result.TestCount)
			}
		}
	}

	if len(allErrs) > 0 {
		return ctx, fmt.Errorf("errors happened during zero downtime tests: %w", errors.Join(allErrs...))
	}

	return ctx, nil
}

func (zd *ZeroDowntimeTestRunner) CleanZeroDowntimeTests(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	for _, tester := range zd.testers {
		log.Printf("Stopping zero downtime tester %s", tester.Name())
		tester.Stop()
	}
	return ctx, nil
}

func fetchIstioIngressGatewayAddress(ctx context.Context, c client.Client) (string, error) {
	istioIngressGatewayNamespaceName := types.NamespacedName{
		Name:      "istio-ingressgateway",
		Namespace: "istio-system",
	}

	var ingressIp string
	var ingressPort int32

	runsOnGardener, err := RunsOnGardener(ctx, c)
	if err != nil {
		return "", err
	}

	if runsOnGardener {
		svc := corev1.Service{}
		if err := c.Get(ctx, istioIngressGatewayNamespaceName, &svc); err != nil {
			return "", err
		}

		if len(svc.Status.LoadBalancer.Ingress) == 0 {
			return "", errors.New("no ingress ip found")
		} else {
			lbIp, err := ip.GetLoadBalancerIp(svc.Status.LoadBalancer.Ingress[0])
			if err != nil {
				return "", err
			}

			ingressIp = lbIp.String()

			for _, port := range svc.Spec.Ports {
				if port.Name == "http2" {
					ingressPort = port.Port
				}
			}
		}
	} else {
		// In case we are not running on Gardener we assume that it's a k3d cluster that has 127.0.0.1 as default address
		ingressIp = "localhost"
		ingressPort = 80
	}

	return fmt.Sprintf("%s:%d", ingressIp, ingressPort), nil
}

type Tester interface {
	Name() string
	Start()
	Stop() []TestResult
}

func NewTester(name string, testFn func() error, numberOfThreads int) Tester {
	return &tester{
		name:            name,
		testFn:          testFn,
		numberOfWorkers: numberOfThreads,
	}
}

type TestResult struct {
	WorkerName string
	TestCount  int
	Err        error
}

type tester struct {
	name            string
	testFn          func() error
	numberOfWorkers int
	resultChans     []chan TestResult
	cancel          func()
	waitGroup       *sync.WaitGroup
}

type worker struct {
	name       string
	test       func() error
	testCount  int
	err        error
	resultChan chan TestResult
}

func (w *worker) doWorkInBackground(ctx context.Context, group *sync.WaitGroup) {
	group.Add(1)
	go func() {
		defer group.Done()
		w.doWork(ctx)
	}()
}

func (w *worker) sendResult() {
	w.resultChan <- TestResult{
		WorkerName: w.name,
		TestCount:  w.testCount,
		Err:        w.err,
	}
	close(w.resultChan)
}

func (w *worker) doWork(ctx context.Context) {
	defer w.sendResult()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			w.testCount++
			err := w.test()
			if err != nil {
				w.err = fmt.Errorf("test %d done by worker %s failed with error %v", w.testCount, w.name, err)
				return
			}
		}
	}
}

func (t *tester) Name() string {
	return t.name
}

func (t *tester) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.waitGroup = &sync.WaitGroup{}
	t.resultChans = make([]chan TestResult, t.numberOfWorkers)

	for i := 0; i < t.numberOfWorkers; i++ {
		t.resultChans[i] = make(chan TestResult, 1)
		w := worker{
			name:       fmt.Sprintf("%s-%d", t.name, i),
			test:       t.testFn,
			resultChan: t.resultChans[i],
		}

		w.doWorkInBackground(ctx, t.waitGroup)
	}
}

func (t *tester) Stop() []TestResult {
	results := make([]TestResult, 0)
	t.cancel()
	t.waitGroup.Wait()
	for _, resultChan := range t.resultChans {
		result := <-resultChan
		results = append(results, result)
	}
	return results
}

func RunsOnGardener(ctx context.Context, k8sClient client.Client) (bool, error) {
	cmShootInfo := corev1.ConfigMap{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "kube-system", Name: "shoot-info"}, &cmShootInfo)

	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	return true, err
}
