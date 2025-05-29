package webhooks

import (
	"context"
	"fmt"
	"time"

	"github.com/thoas/go-funk"
	"istio.io/api/label"
	v1 "k8s.io/api/admissionregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/avast/retry-go"
	"istio.io/istio/istioctl/pkg/tag"
)

const (
	retriesCount        = 5
	delayBetweenRetries = 5 * time.Second
	IstioTagLabel       = "istio.io/tag"
)

func GetDeactivatedLabel() map[string]string {
	return map[string]string{
		"istio.io/deactivated": "never-match",
	}
}

// DeleteConflictedDefaultTag deletes conflicted tagged MutatingWebhookConfiguration, if it exists and if the default revision MutatingWebhookConfiguration is not deactivated by Istio installation logic.
func DeleteConflictedDefaultTag(ctx context.Context, kubeClient client.Client) error {
	retryOpts := []retry.Option{
		retry.Delay(delayBetweenRetries),
		retry.Attempts(uint(retriesCount)),
		retry.DelayType(retry.FixedDelay),
	}

	err := retry.Do(func() error {
		webhooks, err := getWebhooksWithTag(ctx, kubeClient, tag.DefaultRevisionName)
		if err != nil {
			return err
		}
		// As the default revision is not deactivated and handles the injection, we are safe to delete all other webhook configurations that were created during the failed installation.
		if !isDefaultRevisionDeactivated(ctx, kubeClient) && len(webhooks) > 0 {
			for _, wh := range webhooks {
				apiErr := kubeClient.Delete(ctx, &wh)
				if apiErr != nil {
					return apiErr
				}
			}
		}

		return nil
	}, retryOpts...)

	if err != nil {
		return err
	}

	return nil
}

// getWebhooksWithTag returns webhooks tagged with istio.io/tag=<tag>.
// This implementation is the same as in istioctl/pkg/tag/util package, but migrated to controller runtime client.
func getWebhooksWithTag(ctx context.Context, kubeClient client.Client, tag string) ([]v1.MutatingWebhookConfiguration, error) {
	var webhooks v1.MutatingWebhookConfigurationList
	err := kubeClient.List(ctx, &webhooks, client.MatchingLabels{
		IstioTagLabel: tag,
	})
	if err != nil {
		return nil, err
	}
	return webhooks.Items, nil
}

// getWebhooksWithRevision returns webhooks tagged with istio.io/rev=<rev> and NOT TAGGED with istio.io/tag.
// This implementation is the same as in istioctl/pkg/tag/util package, but migrated to controller runtime client.
func getWebhooksWithRevision(ctx context.Context, kubeClient client.Client, rev string) ([]v1.MutatingWebhookConfiguration, error) {
	var webhooks v1.MutatingWebhookConfigurationList
	err := kubeClient.List(ctx, &webhooks, client.MatchingLabels{
		label.IoIstioRev.Name: rev,
	})
	if err != nil {
		return nil, err
	}

	// this cast is tricky. TODO: we should probably use something else and avoid casting result of a function this way
	webhooks.Items, _ = funk.Filter(webhooks.Items,
		func(w v1.MutatingWebhookConfiguration) bool {
			_, ok := w.Labels[IstioTagLabel]
			return !ok
		}).([]v1.MutatingWebhookConfiguration)

	return webhooks.Items, nil
}

func isDefaultRevisionDeactivated(ctx context.Context, client client.Client) bool {
	// This will contain only webhook  with istio.io/rev=default and without istio.io/tag label - the default one, applied from Helm
	mwcs, err := getWebhooksWithRevision(ctx, client, tag.DefaultRevisionName)
	if err != nil {
		return true
	}

	if len(mwcs) == 0 {
		return true
	}

	for _, mwc := range mwcs {
		for _, wh := range mwc.Webhooks {
			if fmt.Sprint(wh.NamespaceSelector.MatchLabels) == fmt.Sprint(GetDeactivatedLabel()) && fmt.Sprint(wh.ObjectSelector.MatchLabels) == fmt.Sprint(GetDeactivatedLabel()) {
				return true
			}
		}
	}

	return false
}
