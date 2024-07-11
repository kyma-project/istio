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
		// given
		r1 := &restarterMock{}
		r2 := &restarterMock{}

		// when
		err, requeue := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r1, r2})

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(requeue).To(BeFalse())
	})

	It("should return nil if no restarters are provided", func() {
		// when
		err, requeue := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, nil)

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(requeue).To(BeFalse())
	})

	It("should invoke Restart", func() {
		// given
		r := &restarterMock{}

		// when
		err, requeue := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r})

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(requeue).To(BeFalse())
		Expect(r.RestartCalled()).Should(BeTrue())
	})

	It("should return error if Restart fails", func() {
		// given
		r := &restarterMock{err: described_errors.NewDescribedError(errors.New("restart error"), "")}

		// when
		err, requeue := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r})

		// then
		Expect(err).Should(MatchError("restart error"))
		Expect(requeue).To(BeFalse())
	})

	It("should return error with Error level when restarters return Error and Warning level errors", func() {
		// given
		r1 := &restarterMock{err: described_errors.NewDescribedError(errors.New("restart error"), "")}
		r2 := &restarterMock{err: described_errors.NewDescribedError(errors.New("restart warning"), "").SetWarning()}

		// when
		err, requeue := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r1, r2})

		// then
		Expect(err).Should(MatchError("restart error"))
		Expect(requeue).To(BeFalse())
	})

	It("should respect requeue condition if one of the restarters return it", func() {
		// given
		r1 := &restarterMock{requeue: false}
		r2 := &restarterMock{requeue: true}

		// when
		err, requeue := restarter.Restart(context.Background(), &operatorv1alpha2.Istio{}, []restarter.Restarter{r1, r2})

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(r1.RestartCalled()).Should(BeTrue())
		Expect(r2.RestartCalled()).Should(BeTrue())
		Expect(requeue).To(BeTrue())
	})
})

type restarterMock struct {
	err       described_errors.DescribedError
	requeue   bool
	restarted bool
}

func (i *restarterMock) RestartCalled() bool {
	return i.restarted
}

func (i *restarterMock) Restart(_ context.Context, _ *operatorv1alpha2.Istio) (described_errors.DescribedError, bool) {
	i.restarted = true
	return i.err, i.requeue
}
