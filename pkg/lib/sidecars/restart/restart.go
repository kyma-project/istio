package restart

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ownerReferenceNotFoundMessage = "pod sidecar could not be updated because OwnerReferences was not found."
	ownedByJobMessage             = "pod sidecar could not be updated because it is owned by a Job."
)

type ActionRestarter interface {
	RestartAction(ctx context.Context, podList *v1.PodList, failOnError bool) ([]RestartWarning, error)
}

type ActionRestart struct {
	k8sClient client.Client
	logger    *logr.Logger
}

func NewActionRestarter(c client.Client, logger *logr.Logger) *ActionRestart {
	return &ActionRestart{
		k8sClient: c,
		logger:    logger,
	}
}

type RestartWarning struct {
	Name, Namespace, Kind, Message string
}

func newRestartWarning(o actionObject, message string) RestartWarning {
	return RestartWarning{
		Name:      o.Name,
		Namespace: o.Namespace,
		Kind:      o.Kind,
		Message:   message,
	}
}

func (s *ActionRestart) RestartAction(ctx context.Context, podList *v1.PodList, failOnError bool) ([]RestartWarning, error) {
	warnings := make([]RestartWarning, 0)
	processedActionObjects := make(map[string]bool)

	for _, pod := range podList.Items {
		action, err := restartActionFactory(ctx, s.k8sClient, pod)
		if err != nil {
			s.logger.Error(err, "pod", action.object.getKey(), "Creating pod restart action failed")
			if failOnError {
				return warnings, fmt.Errorf("creating pod restart action failed: %w", err)
			}
			continue
		}

		// We want to avoid performing the same action multiple times for a parent if it contains multiple pods that need to be restarted.
		if _, exists := processedActionObjects[action.object.getKey()]; !exists {
			currentWarnings, err := action.run(ctx, s.k8sClient, action.object, s.logger)
			if err != nil {
				s.logger.Error(err, "pod", action.object.getKey(), "Running pod restart action failed")
				if failOnError {
					return warnings, fmt.Errorf("running pod restart action failed: %w", err)
				}
			}
			warnings = append(warnings, currentWarnings...)
			processedActionObjects[action.object.getKey()] = true
		}

	}

	return warnings, nil
}
