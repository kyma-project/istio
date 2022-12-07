package restart

import v1 "k8s.io/api/core/v1"

type warningAction struct {
	object  actionObject
	message string
}

func (r warningAction) run() ([]RestartWarning, error) {
	return []RestartWarning{newRestartWarning(r.object, r.message)}, nil

}

func newOwnerNotFoundAction(pod v1.Pod) warningAction {
	return warningAction{object: newWarningActionObject(pod), message: ownerReferenceNotFoundMessage}
}

func newOwnedByJobAction(pod v1.Pod) warningAction {
	return warningAction{object: newWarningActionObject(pod), message: ownedByJobMessage}
}

func newWarningActionObject(pod v1.Pod) actionObject {
	return actionObject{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Kind:      pod.Kind,
	}
}
