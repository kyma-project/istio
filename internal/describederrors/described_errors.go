package describederrors

import (
	"github.com/pkg/errors"
)

type Level int

const (
	Error   Level = 0
	Warning Level = 1
)

// DescribedError wraps standard golang error with additional description to be set on Istio CR Status.
type DescribedError interface {
	Description() string
	Error() string
	Level() Level
	ShouldSetCondition() bool
	hasHigherSeverityThan(err DescribedError) bool
}

type DefaultDescribedError struct {
	err          error
	description  string
	wrapError    bool
	level        Level
	setCondition bool
}

func NewDescribedError(err error, description string) DefaultDescribedError {
	return DefaultDescribedError{
		err:          err,
		description:  description,
		wrapError:    true,
		level:        Error,
		setCondition: true,
	}
}

func (d DefaultDescribedError) DisableErrorWrap() DefaultDescribedError {
	d.wrapError = false
	return d
}

func (d DefaultDescribedError) SetWarning() DefaultDescribedError {
	d.level = Warning
	return d
}

func (d DefaultDescribedError) SetCondition(setCondition bool) DefaultDescribedError {
	d.setCondition = setCondition
	return d
}

func (d DefaultDescribedError) Description() string {
	if d.wrapError {
		return errors.Wrap(d.err, d.description).Error()
	}
	return d.description
}

func (d DefaultDescribedError) Error() string {
	return d.err.Error()
}

func (d DefaultDescribedError) Level() Level {
	return d.level
}

func (d DefaultDescribedError) ShouldSetCondition() bool {
	return d.setCondition
}

// GetMostSevereErr returns the most severe error from the list of errors.
func GetMostSevereErr(errs []DescribedError) DescribedError {
	var candidate DescribedError
	for _, err := range errs {
		if err == nil {
			continue
		}
		if candidate == nil || err.hasHigherSeverityThan(candidate) {
			candidate = err
		}
	}
	return candidate
}

func (d DefaultDescribedError) hasHigherSeverityThan(err DescribedError) bool {
	// This checks if the receiver error has a higher level than the parameter. This might be counterintuitive, but it's correct because
	// levels are ordered from the most severe to the least severe. So, if the current error has a lower level than the candidate, it's more severe.
	return d.Level() < err.Level()
}
