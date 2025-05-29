package describederrors_test

import (
	"github.com/kyma-project/istio/operator/internal/describederrors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("GetMostSevereErr", func() {

	It("should return error with the highest severity", func() {
		err := describederrors.GetMostSevereErr([]describederrors.DescribedError{
			describederrors.NewDescribedError(errors.New("warning"), "").SetWarning(),
			describederrors.NewDescribedError(errors.New("error"), ""),
			describederrors.NewDescribedError(errors.New("warning"), "").SetWarning(),
		})
		Expect(err).Should(MatchError("error"))

		err = describederrors.GetMostSevereErr([]describederrors.DescribedError{
			describederrors.NewDescribedError(errors.New("error"), ""),
			describederrors.NewDescribedError(errors.New("warning"), "").SetWarning(),
			describederrors.NewDescribedError(errors.New("warning"), "").SetWarning(),
		})
		Expect(err).Should(MatchError("error"))

		err = describederrors.GetMostSevereErr([]describederrors.DescribedError{
			describederrors.NewDescribedError(errors.New("warning"), "").SetWarning(),
			describederrors.NewDescribedError(errors.New("warning"), "").SetWarning(),
			describederrors.NewDescribedError(errors.New("error"), ""),
		})
		Expect(err).Should(MatchError("error"))

		err = describederrors.GetMostSevereErr([]describederrors.DescribedError{
			describederrors.NewDescribedError(errors.New("warning"), "").SetWarning(),
			nil,
			describederrors.NewDescribedError(errors.New("warning"), "").SetWarning(),
			nil,
			describederrors.NewDescribedError(errors.New("error"), ""),
			describederrors.NewDescribedError(errors.New("warning"), "").SetWarning(),
		})
		Expect(err).Should(MatchError("error"))
	})

	It("should return nil if no errors are given", func() {
		err := describederrors.GetMostSevereErr([]describederrors.DescribedError{})

		Expect(err).Should(BeNil())
	})

	It("should return nil if array of errors is filled with nils and errors", func() {
		mockErr := describederrors.NewDescribedError(errors.New("error"), "")
		err := describederrors.GetMostSevereErr([]describederrors.DescribedError{nil, mockErr, nil})

		Expect(err).Should(MatchError("error"))
	})
	It("should return nil if array of errors is filled with nils", func() {
		err := describederrors.GetMostSevereErr([]describederrors.DescribedError{nil, nil, nil})
		Expect(err).Should(BeNil())
	})
})
