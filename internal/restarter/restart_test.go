package restarter_test

import (
	"context"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/restarter"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Restart", func() {

	It("should return nil if no restarters fail", func() {
		r1 := &restarterMock{}
		r2 := &restarterMock{}

		Expect(restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r1, r2})).Should(Succeed())

	})

	It("should return nil if no restarters are provided", func() {
		Expect(restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, nil)).Should(Succeed())
	})

	It("should invoke Restart", func() {
		r := &restarterMock{}

		Expect(restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r})).Should(Succeed())

		Expect(r.RestartCalled()).Should(BeTrue())
	})

	It("should return error if Restart fails", func() {
		r := &restarterMock{err: described_errors.NewDescribedError(errors.New("restart error"), "")}

		Expect(restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r})).Should(MatchError("restart error"))
	})

	It("should return error with Error level when restarters return Error and Warning level errors", func() {
		r1 := &restarterMock{err: described_errors.NewDescribedError(errors.New("restart error"), "")}
		r2 := &restarterMock{err: described_errors.NewDescribedError(errors.New("restart warning"), "").SetWarning()}

		err := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r1, r2})

		Expect(err).Should(MatchError("restart error"))
	})
})

type restarterMock struct {
	err       described_errors.DescribedError
	restarted bool
}

func (i *restarterMock) RestartCalled() bool {
	return i.restarted
}

func (i *restarterMock) Restart(_ context.Context, _ *operatorv1alpha2.Istio) described_errors.DescribedError {
	i.restarted = true
	return i.err
}
