package controllers

import (
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/onsi/ginkgo/v2/types"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	securityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	v1 "k8s.io/api/core/v1"
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
	return fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(objects...).WithStatusSubresource(objects...).Build()
}

func getTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	Expect(operatorv1alpha1.AddToScheme(scheme)).Should(Succeed())
	Expect(v1.AddToScheme(scheme)).Should(Succeed())
	Expect(v1alpha3.AddToScheme(scheme)).Should(Succeed())
	Expect(v1beta1.AddToScheme(scheme)).Should(Succeed())
	Expect(securityv1beta1.AddToScheme(scheme)).Should(Succeed())

	return scheme
}
