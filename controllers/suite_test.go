package controllers

import (
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/onsi/ginkgo/v2/types"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("controller-suite", report)
})

func createFakeClient(objects ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(objects...).Build()
}

func getTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	Expect(operatorv1alpha1.AddToScheme(scheme)).Should(Succeed())

	return scheme
}
