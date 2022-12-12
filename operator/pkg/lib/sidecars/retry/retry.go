package retry

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

func IsRetryError(err error) bool {
	if errors.IsTooManyRequests(err) ||
		errors.IsServerTimeout(err) ||
		errors.IsTimeout(err) ||
		errors.IsServiceUnavailable(err) ||
		errors.IsConflict(err) ||
		errors.IsNotFound(err) {
		return true
	}
	return false
}

func RetryOnError(backoff wait.Backoff, fn func() error) error {
	return retry.OnError(backoff, IsRetryError, fn)
}

func RetryOnConflict(backoff wait.Backoff, fn func() error) error {
	return retry.RetryOnConflict(backoff, fn)
}
