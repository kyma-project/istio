package restart

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const rolloutTimeoutMessage = "pod could not be rolled out by resource owner's controller."

func newRolloutAction(object actionObject) restartAction {
	return restartAction{
		run:    rolloutRun,
		object: object,
	}
}

func rolloutRun(ctx context.Context, client client.Client, object actionObject) ([]RestartWarning, error) {
	return []RestartWarning{newRestartWarning(object, rolloutTimeoutMessage)}, nil
}
