package described_errors

import "github.com/pkg/errors"

// DescribedError wraps standard golang error with additional description to be set on Istio CR Status
type DescribedError interface {
	Description() string
	Error() string
}

type DefaultDescribedError struct {
	err         error
	description string
	wrapError   bool
}

func NewDescribedError(err error, description string) DefaultDescribedError {
	return DefaultDescribedError{
		err:         err,
		description: description,
		wrapError:   true,
	}
}

func (d DefaultDescribedError) DisableErrorWrap() DefaultDescribedError {
	d.wrapError = false
	return d
}

func (d DefaultDescribedError) Description() string {
	if d.wrapError {
		return errors.Wrap(d.err, d.description).Error()
	} else {
		return d.description
	}
}

func (d DefaultDescribedError) Error() string {
	return d.err.Error()
}
