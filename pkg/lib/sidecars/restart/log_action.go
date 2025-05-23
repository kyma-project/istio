package restart

import (
	"context"

	"github.com/go-logr/logr"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type logAction struct {
	message string
}

func (r logAction) run(_ context.Context, _ client.Client, object actionObject, l *logr.Logger) ([]Warning, error) {
	l.Info(r.message, "kind", object.Kind, "name", object.Name, "namespace", object.Namespace)
	return []Warning{}, nil
}
