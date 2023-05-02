package integration

import (
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type Config struct {
	ClientTimeout   time.Duration `envconfig:"TEST_CLIENT_TIMEOUT,default=10s"`
	ReqTimeout      time.Duration `envconfig:"TEST_REQUEST_TIMEOUT,default=180s"`
	ReqDelay        time.Duration `envconfig:"TEST_REQUEST_DELAY,default=5s"`
	TestConcurrency int           `envconfig:"TEST_CONCURRENCY,default=1"`
}

const exportResultVar string = "EXPORT_RESULT"

var (
	retryOpts []retry.Option
	k8sClient client.Client
	conf      Config
)

var goDogOpts = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "pretty",
}

var CRDGroupVersionKind = schema.GroupVersionKind{
	Group:   "apiextensions.k8s.io",
	Version: "v1",
	Kind:    "CustomResourceDefinition",
}

const crdListPath string = "manifests/crd_list.yaml"

type godogResourceMapping int

func (k godogResourceMapping) String() string {
	switch k {
	case DaemonSet:
		return "DaemonSet"
	case Deployment:
		return "Deployment"
	case IstioCR:
		return "Istio CR"
	}
	panic(fmt.Errorf("%#v has unimplemented String() method", k))
}

const (
	DaemonSet godogResourceMapping = iota
	Deployment
	IstioCR
)
