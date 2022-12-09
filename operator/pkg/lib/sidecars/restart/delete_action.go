package restart

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newDeleteAction(object actionObject) restartAction {
	return restartAction{
		object: object,
		run:    deleteRun,
	}
}

func deleteRun(ctx context.Context, client client.Client, object actionObject) ([]RestartWarning, error) {
	return nil, client.Delete(ctx, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      object.Name,
			Namespace: object.Namespace,
		},
	})
}
