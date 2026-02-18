//go:build experimental

package clusterconfig_test

import (
	"context"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DualStack", func() {
	It("should be enabled when the controller is running in experimental mode,"+
		"and the cluster has kyma-runtime-config CM with dualstack enabled", func() {
		client := createFakeClient(createKymaRuntimeConfigWithDualStack(true))

		//when
		ds, err := clusterconfig.IsDualStackEnabled(context.Background(), client)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(ds).To(Equal(true))
	})
})
