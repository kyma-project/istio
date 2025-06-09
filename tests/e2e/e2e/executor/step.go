package executor

import (
	"context"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Step interface {
	Description() string
	Execute(context.Context, client.Client, *log.Logger) error
	Cleanup(context.Context, client.Client) error
}
