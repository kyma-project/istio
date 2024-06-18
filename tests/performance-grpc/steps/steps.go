package steps

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/testcontext"
	"istio.io/istio/pkg/test/util/tmpl"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"os"
	"sigs.k8s.io/yaml"
)

const loadTestingNamespace = "load-testing"

type TemplatedPerformanceJob struct {
	templatedValues map[string]string
	jobName         string
}

func (t *TemplatedPerformanceJob) SetTemplateValue(key, value string) error {
	if t.templatedValues == nil {
		t.templatedValues = make(map[string]string)
	}

	t.templatedValues[key] = value
	return nil
}

func getRequestHost() string {
	kymaDomain := os.Getenv("KYMA_DOMAIN")
	if kymaDomain == "" {
		print("KYMA_DOMAIN env variable is not set, using default address set in the Job template")
		return ""
	}

	return fmt.Sprintf("%s.%s:443", "hello", kymaDomain)
}

func getRequestAuthority() string {
	kymaDomain := os.Getenv("KYMA_DOMAIN")
	if kymaDomain == "" {
		print("KYMA_DOMAIN env variable is not set, using default address set in the Job template")
		return ""
	}

	return fmt.Sprintf("%s.%s", "hello", kymaDomain)
}

//go:embed job.yaml
var jobTemplate string

func (t *TemplatedPerformanceJob) ExecutePerformanceTest(ctx context.Context) error {
	template, err := tmpl.Parse(jobTemplate)
	if err != nil {
		return err
	}

	name := string(uuid.NewUUID())
	t.templatedValues["HTMLReportPath"] = fmt.Sprintf("%s.html", name)

	useExternalIngressGatewayHost := os.Getenv("USE_EXTERNAL_INGRESS_GATEWAY_HOST") == "true"
	if useExternalIngressGatewayHost {
		host := getRequestHost()
		if host != "" {
			t.templatedValues["Address"] = host
		}
	} else {
		authority := getRequestAuthority()
		if authority != "" {
			t.templatedValues["Authority"] = authority
		}
	}

	jobYaml, err := tmpl.Execute(template, t.templatedValues)
	if err != nil {
		return err
	}

	var job batchv1.Job

	err = yaml.Unmarshal([]byte(jobYaml), &job)
	if err != nil {
		return err
	}

	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	job.Namespace = loadTestingNamespace
	job.Name = name
	t.jobName = name

	err = k8sClient.Create(ctx, &job)

	return err
}

func (t *TemplatedPerformanceJob) TestShouldRunSuccessfully(ctx context.Context) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	var job batchv1.Job

	return retry.Do(func() error {
		err = k8sClient.Get(ctx, types.NamespacedName{Name: t.jobName, Namespace: loadTestingNamespace}, &job)
		if err != nil {
			return err
		}

		if job.Status.Succeeded != 1 {
			return fmt.Errorf("job %s failed", t.jobName)
		}
		return nil
	}, testcontext.GetRetryOpts()...)
}
