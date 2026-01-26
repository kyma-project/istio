package istioassert

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
)

type StatusAssertOptions struct {
	ExpectedState       *v1alpha2.State
	ExpectedConditions  []ExpectedCondition
	ExpectedDescription []string
	Timeout             time.Duration
	Interval            time.Duration
}

type ExpectedCondition struct {
	Type    v1alpha2.ConditionType
	Status  metav1.ConditionStatus
	Reason  v1alpha2.ConditionReason
	Message string // optional - if empty, won't check message
}

type StatusAssertOption func(*StatusAssertOptions)

// WithExpectedState sets the expected Istio CR state (e.g., Ready, Warning, Error)
func WithExpectedState(state v1alpha2.State) StatusAssertOption {
	return func(o *StatusAssertOptions) {
		o.ExpectedState = &state
	}
}

// WithExpectedCondition adds an expected condition to check
func WithExpectedCondition(condType v1alpha2.ConditionType, status metav1.ConditionStatus, reason v1alpha2.ConditionReason) StatusAssertOption {
	return func(o *StatusAssertOptions) {
		o.ExpectedConditions = append(o.ExpectedConditions, ExpectedCondition{
			Type:   condType,
			Status: status,
			Reason: reason,
		})
	}
}

// WithExpectedConditionMessage adds an expected condition with a specific message to check
func WithExpectedConditionMessage(condType v1alpha2.ConditionType, status metav1.ConditionStatus, reason v1alpha2.ConditionReason, message string) StatusAssertOption {
	return func(o *StatusAssertOptions) {
		o.ExpectedConditions = append(o.ExpectedConditions, ExpectedCondition{
			Type:    condType,
			Status:  status,
			Reason:  reason,
			Message: message,
		})
	}
}

// WithExpectedDescriptionContaining adds expected description content to check
func WithExpectedDescriptionContaining(contains ...string) StatusAssertOption {
	return func(o *StatusAssertOptions) {
		o.ExpectedDescription = append(o.ExpectedDescription, contains...)
	}
}

// WithTimeout sets the timeout for the assertion
func WithTimeout(timeout time.Duration) StatusAssertOption {
	return func(o *StatusAssertOptions) {
		o.Timeout = timeout
	}
}

// WithInterval sets the check interval for the assertion
func WithInterval(interval time.Duration) StatusAssertOption {
	return func(o *StatusAssertOptions) {
		o.Interval = interval
	}
}

// AssertIstioStatus asserts that an Istio CR reaches the expected status conditions
func AssertIstioStatus(t *testing.T, c *resources.Resources, istioCR *v1alpha2.Istio, opts ...StatusAssertOption) {
	t.Helper()

	options := &StatusAssertOptions{
		Timeout:  30 * time.Second,
		Interval: 2 * time.Second,
	}

	for _, opt := range opts {
		opt(options)
	}

	err := wait.For(func(ctx context.Context) (bool, error) {
		istioObj := &v1alpha2.Istio{}
		err := c.Get(t.Context(), istioCR.GetName(), istioCR.GetNamespace(), istioObj)
		if err != nil {
			t.Logf("Failed to get Istio CR: %v", err)
			return false, nil
		}
		status := &istioObj.Status

		// Check state if specified
		if options.ExpectedState != nil && status.State != *options.ExpectedState {
			t.Logf("Expected state %s, got %s", *options.ExpectedState, status.State)
			return false, nil
		}

		// Check conditions
		if len(options.ExpectedConditions) > 0 {
			if status.Conditions == nil {
				t.Logf("Expected conditions but status.Conditions is nil")
				return false, nil
			}

			for _, expected := range options.ExpectedConditions {
				condition := findCondition(*status.Conditions, expected.Type)
				if condition == nil {
					t.Logf("Condition %s not found", expected.Type)
					return false, nil
				}

				if condition.Status != expected.Status {
					t.Logf("Condition %s: expected status %s, got %s", expected.Type, expected.Status, condition.Status)
					return false, nil
				}

				if condition.Reason != string(expected.Reason) {
					t.Logf("Condition %s: expected reason %s, got %s", expected.Type, expected.Reason, condition.Reason)
					return false, nil
				}

				if expected.Message != "" && condition.Message != expected.Message {
					t.Logf("Condition %s: expected message %q, got %q", expected.Type, expected.Message, condition.Message)
					return false, nil
				}
			}
		}

		// Check description contains expected strings
		for _, expected := range options.ExpectedDescription {
			if !strings.Contains(status.Description, expected) {
				t.Logf("Expected description to contain %q, got: %s", expected, status.Description)
				return false, nil
			}
		}

		return true, nil
	}, wait.WithTimeout(options.Timeout), wait.WithInterval(options.Interval))

	require.NoError(t, err, "Istio status assertion failed")
}

// AssertReadyStatus is a convenience function to assert Istio CR is in Ready state with ReconcileSucceeded condition
func AssertReadyStatus(t *testing.T, c *resources.Resources, istioCR *v1alpha2.Istio, opts ...StatusAssertOption) {
	t.Helper()
	opts = append([]StatusAssertOption{
		WithExpectedState(v1alpha2.Ready),
		WithExpectedCondition(v1alpha2.ConditionTypeReady, metav1.ConditionTrue, v1alpha2.ConditionReasonReconcileSucceeded),
	}, opts...)
	AssertIstioStatus(t, c, istioCR, opts...)
}

// AssertErrorStatus is a convenience function to assert Istio CR is in Error state
func AssertErrorStatus(t *testing.T, c *resources.Resources, istioCR *v1alpha2.Istio, opts ...StatusAssertOption) {
	t.Helper()
	opts = append([]StatusAssertOption{
		WithExpectedState(v1alpha2.Error),
	}, opts...)
	AssertIstioStatus(t, c, istioCR, opts...)
}

// AssertWarningStatus is a convenience function to assert Istio CR is in Warning state
func AssertWarningStatus(t *testing.T, c *resources.Resources, istioCR *v1alpha2.Istio, opts ...StatusAssertOption) {
	t.Helper()
	opts = append([]StatusAssertOption{
		WithExpectedState(v1alpha2.Warning),
	}, opts...)
	AssertIstioStatus(t, c, istioCR, opts...)
}

// findCondition finds a condition by type in the conditions list
func findCondition(conditions []metav1.Condition, condType v1alpha2.ConditionType) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == string(condType) {
			return &conditions[i]
		}
	}
	return nil
}

// AssertIstioNamespaceExists asserts that the istio-system namespace exists
func AssertIstioNamespaceExists(t *testing.T, c *resources.Resources) error {
	t.Helper()

	istioNs := &v1.Namespace{}
	return c.Get(t.Context(), "istio-system", "", istioNs)
}

// AssertIstioNamespaceDeleted waits for the istio-system namespace to be deleted
func AssertIstioNamespaceDeleted(t *testing.T, c *resources.Resources, timeout time.Duration) error {
	t.Helper()

	istioNs := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "istio-system",
		},
	}
	return wait.For(conditions.New(c).ResourceDeleted(istioNs), wait.WithTimeout(timeout))
}
