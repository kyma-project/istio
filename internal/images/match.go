package images

import v1 "k8s.io/api/core/v1"

func (r Image) MatchesImageIn(container v1.Container) bool {
	return container.Image == r.String()
}
