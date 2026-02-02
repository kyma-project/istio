package images

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
)

type Image struct {
	Registry string
	Name     string
	Tag      string
}

type Images struct {
	Registry string
	Tag      string

	Pilot      Image
	InstallCNI Image
	ProxyV2    Image
	Ztunnel    Image
}

func NewImage(image string) (Image, error) {
	return parseImage(image)
}

func (i Image) String() string {
	if i.Registry == "" {
		return fmt.Sprintf("%s:%s", i.Name, i.Tag)
	}

	if i.Name == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s:%s", i.Registry, i.Name, i.Tag)
}

func (i Image) GetName() string {
	return i.Name
}

func (i Image) GetHub() string {
	return i.Registry
}

func (i Image) GetTag() string {
	return i.Tag
}

func (i Image) GetTagWithoutDigest() string {
	return removeDigestSuffix(i.Tag)
}

// MatchesImageInContainer checks if the image matches the container's image by comparing the full image string.
func (i Image) MatchesImageInContainer(container v1.Container) bool {
	return i.String() == container.Image
}

// RemoveDigestSuffix removes the digest suffix if present (format: @sha256:...)
func removeDigestSuffix(image string) string {
	if idx := strings.Index(image, "@"); idx != -1 {
		return image[:idx]
	}
	return image
}
