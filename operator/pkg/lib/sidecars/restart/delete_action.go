package restart

import v1 "k8s.io/api/core/v1"

type deleteAction struct {
	object actionObject
}

func newDeleteAction(pod v1.Pod) deleteAction {
	return deleteAction{}
}

func newDeleteActionObject(pod v1.Pod) actionObject {
	return actionObject{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Kind:      pod.Kind,
	}
}
