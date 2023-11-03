package clusterconfig_test

import (
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
	"time"
)

var _ = Describe("Hyperscaler", func() {
	Context("IsHyperscalerAWS", func() {
		It("should be true if hyperscaler is aws", func() {
			// given
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			hc := &clusterconfig.HyperscalerClient{
				ts.Client(),
				ts.URL,
			}
			// when
			isAws := hc.IsAws()

			// then
			Expect(isAws).To(BeTrue())
		})

		It("should be false if hyperscaler is not aws", func() {
			// given
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}))
			defer ts.Close()

			hc := &clusterconfig.HyperscalerClient{
				ts.Client(),
				ts.URL,
			}

			// when
			isAws := hc.IsAws()

			// then
			Expect(isAws).To(BeFalse())
		})

		It("should be false if metadata server is unreachable", func() {
			// given
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}))
			ts.Config.ReadTimeout = 1 * time.Millisecond
			// simulates non-reachable server
			ts.Close()

			hc := &clusterconfig.HyperscalerClient{
				ts.Client(),
				ts.URL,
			}

			// when
			isAws := hc.IsAws()

			// then
			Expect(isAws).To(BeFalse())
		})

	})
})
