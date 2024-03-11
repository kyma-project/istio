package described_errors_test

import (
	"github.com/kyma-project/istio/operator/internal/described_errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("GetMostSevereErr", func() {

	It("should return error with the highest severity", func() {
		err1 := described_errors.NewDescribedError(errors.New("error"), "")
		err2 := described_errors.NewDescribedError(errors.New("warning"), "").SetWarning()

		err := described_errors.GetMostSevereErr([]described_errors.DescribedError{err1, err2})

		Expect(err).Should(MatchError("error"))
	})

	It("should return nil if no errors are given", func() {
		err := described_errors.GetMostSevereErr([]described_errors.DescribedError{})

		Expect(err).Should(BeNil())
	})

	It("should return nil if array of errors is filled with nils and errors", func() {
		mockErr := described_errors.NewDescribedError(errors.New("error"), "")
		err := described_errors.GetMostSevereErr([]described_errors.DescribedError{nil, mockErr, nil})

		Expect(err).Should(MatchError("error"))
	})
	It("should return nil if array of errors is filled with nils", func() {
		err := described_errors.GetMostSevereErr([]described_errors.DescribedError{nil, nil, nil})
		Expect(err).Should(BeNil())
	})
})
