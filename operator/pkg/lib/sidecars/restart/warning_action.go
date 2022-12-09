package restart

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type warningAction struct {
	message string
}

func (r warningAction) run(ctx context.Context, client client.Client, object actionObject) ([]RestartWarning, error) {
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

func newWarningActionObject(pod v1.Pod) actionObject {
	return actionObject{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Kind:      pod.Kind,
	}
}
