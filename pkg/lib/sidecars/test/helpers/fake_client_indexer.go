package helpers

import (
	"fmt"
	"k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FakePodStatusPhaseIndexer(object client.Object) []string {
	p, ok := object.(*v1.Pod)
	if !ok {
		panic(fmt.Errorf("indexer function for type %T's status.phase field received object of type %T", v1.Pod{}, object))
	}
	return []string{string(p.Status.Phase)}
}
