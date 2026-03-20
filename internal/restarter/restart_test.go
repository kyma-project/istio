package restarter_test

import (
	"context"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/describederrors"
	"github.com/kyma-project/istio/operator/internal/restarter"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Restart", func() {
	It("should return nil if no restarters fail", func() {
		// given
		r1 := &restarterMock{}
		r2 := &restarterMock{}

		// when
		err := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r1, r2})

		// then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return nil if no restarters are provided", func() {
		// when
		err := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, nil)

		// then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should invoke Restart", func() {
		// given
		r := &restarterMock{}

		// when
		err := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r})

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(r.RestartCalled()).Should(BeTrue())
	})

	It("should return error if Restart fails", func() {
		// given
		r := &restarterMock{err: describederrors.NewDescribedError(errors.New("restart error"), "")}

		// when
		err := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r})

		// then
		Expect(err).Should(MatchError("restart error"))
	})

	It("should return error with Error level when restarters return Error and Warning level errors", func() {
		// given
		r1 := &restarterMock{err: describederrors.NewDescribedError(errors.New("restart error"), "")}
		r2 := &restarterMock{err: describederrors.NewDescribedError(errors.New("restart warning"), "").SetWarning()}

		// when
		err := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r1, r2})

		// then
		Expect(err).Should(MatchError("restart error"))
	})

	It("should invoke all restarters even if one fails", func() {
		// given
		r1 := &restarterMock{}
		r2 := &restarterMock{}

		// when
		err := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r1, r2})

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(r1.RestartCalled()).Should(BeTrue())
		Expect(r2.RestartCalled()).Should(BeTrue())
	})
})

type restarterMock struct {
	err       describederrors.DescribedError
	restarted bool
}

func (i *restarterMock) RestartCalled() bool {
	return i.restarted
}

func (i *restarterMock) Restart(_ context.Context, _ *operatorv1alpha2.Istio) describederrors.DescribedError {
	i.restarted = true
	return i.err
}
