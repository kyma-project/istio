package restart

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ownerReferenceNotFoundMessage = "pod sidecar could not be updated because OwnerReferences was not found."
	ownedByJobMessage             = "pod sidecar could not be updated because it is owned by a Job."
)

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

func Restart(ctx context.Context, c client.Client, podList v1.PodList) ([]RestartWarning, error) {

	warnings := make([]RestartWarning, 0)
	processedActionObjects := make(map[string]bool)

	for _, pod := range podList.Items {
		action, err := restartActionFactory(ctx, c, pod)
		if err != nil {
			return nil, err
		}

		// We want to avoid performing the same action multiple times for a parent if it contains multiple pods that need to be restarted.
		if _, exists := processedActionObjects[action.object.getKey()]; !exists {
			currentWarnings, err := action.run(ctx, c, action.object)
			if err != nil {
				return nil, err
			}
			warnings = append(warnings, currentWarnings...)
			processedActionObjects[action.object.getKey()] = true
		}

	}

	return warnings, nil
}
