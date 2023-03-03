package restart

import (
	"context"
	"github.com/go-logr/logr"

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

func deleteRun(ctx context.Context, client client.Client, object actionObject, logger *logr.Logger) ([]RestartWarning, error) {
	logger.Info("Delete pod due to proxy restart", "name", object.Name, "namespace", object.Namespace)
	return nil, client.Delete(ctx, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      object.Name,
			Namespace: object.Namespace,
		},
	})
}
