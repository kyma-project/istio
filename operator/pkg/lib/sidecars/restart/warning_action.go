package restart

import (
	"context"
	"github.com/go-logr/logr"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type warningAction struct {
	message string
}

func (r warningAction) run(_ context.Context, _ client.Client, object actionObject, _ *logr.Logger) ([]RestartWarning, error) {
	return []RestartWarning{newRestartWarning(object, r.message)}, nil
}

func newOwnerNotFoundAction(pod v1.Pod) restartAction {
	return restartAction{
		object: actionObjectFromPod(pod),
		run:    warningAction{message: ownerReferenceNotFoundMessage}.run,
	}
}

func newOwnedByJobAction(pod v1.Pod) restartAction {
	return restartAction{
		object: actionObjectFromPod(pod),
		run:    warningAction{message: ownedByJobMessage}.run,
	}
}
