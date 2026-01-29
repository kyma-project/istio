package images

import (
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
)

const kymaFipsModeEnabledEnv = "KYMA_FIPS_MODE_ENABLED"

type Image string

type RegistryAndTag struct {
	Registry string
	Tag      string
}

type imageComponents struct {
	registry string
	name     string
	tag      string
}

// parseImage parses an OCI image reference into its components: registry, name, and tag.
// Supports formats like:
//   - registry/namespace/image:tag
//   - registry:port/namespace/image:tag
//   - registry/namespace/image:tag@sha256:digest
//   - registry:port/namespace/image:tag@sha256:digest
func (i Image) parseImage() (imageComponents, error) {
	if i == "" {
		return imageComponents{}, fmt.Errorf("image can not be empty")
	}

	imageStr := string(i)

	// Find the last "/" to separate registry from image name
	lastSlash := strings.LastIndex(imageStr, "/")
	if lastSlash == -1 {
		return imageComponents{}, fmt.Errorf("image %s does not contain a valid format", i)
	}

	registry := imageStr[:lastSlash]
	namePart := imageStr[lastSlash+1:]

	// Find the tag separator ":" in the image name portion
	colonIdx := strings.Index(namePart, ":")
	if colonIdx == -1 {
		return imageComponents{}, fmt.Errorf("image %s does not contain a valid tag", i)
	}

	return imageComponents{
		registry: registry,
		name:     namePart[:colonIdx],
		tag:      namePart[colonIdx+1:],
	}, nil
}

func (i Image) GetHub() (string, error) {
	components, err := i.parseImage()
	if err != nil {
		return "", err
	}
	return components.registry, nil
}

func (i Image) GetTag() (string, error) {
	components, err := i.parseImage()
	if err != nil {
		return "", err
	}
	return components.tag, nil
}

func (i Image) GetName() (string, error) {
	components, err := i.parseImage()
	if err != nil {
		return "", err
	}
	return components.name, nil
}

type Images struct {
	Pilot      Image `env:"pilot,notEmpty"`
	InstallCNI Image `env:"install-cni,notEmpty"`
	ProxyV2    Image `env:"proxyv2,notEmpty"`
	Ztunnel    Image `env:"ztunnel"`
}

type ImagesFips struct {
	Pilot      Image `env:"pilot-fips,notEmpty"`
	InstallCNI Image `env:"install-cni-fips,notEmpty"`
	ProxyV2    Image `env:"proxyv2-fips,notEmpty"`
	Ztunnel    Image `env:"ztunnel-fips"`
}

func GetImages() (*Images, error) {
	kymaFipsModeEnabled := os.Getenv(kymaFipsModeEnabledEnv)
	if kymaFipsModeEnabled == "true" {
		environments, err := env.ParseAs[ImagesFips]()
		if err != nil {
			return nil, fmt.Errorf("missing required environment variables %w", err)
		}
		return (*Images)(&environments), nil
	}

	environments, err := env.ParseAs[Images]()
	if err != nil {
		return nil, fmt.Errorf("missing required environment variables %w", err)
	}

	return &environments, nil
}

func (e *Images) GetImageRegistryAndTag() (RegistryAndTag, error) {
	environments := []Image{e.Pilot, e.InstallCNI, e.ProxyV2}
	if e.Ztunnel != "" {
		environments = append(environments, e.Ztunnel)
	}

	initialHub, err := environments[0].GetHub()
	if err != nil {
		return RegistryAndTag{}, fmt.Errorf("failed to get hub for image %s: %w", environments[0], err)
	}
	initialTag, err := environments[0].GetTag()
	tag := removeDigestSuffix(initialTag)

	if err != nil {
		return RegistryAndTag{}, fmt.Errorf("failed to get tag for image %s: %w", environments[0], err)
	}

	// Ensure that all required images are from the same hub and have the same version tag
	for _, image := range environments {
		currentHub, err := image.GetHub()
		if err != nil {
			return RegistryAndTag{}, fmt.Errorf("failed to get hub for image %s: %w", image, err)
		}
		if currentHub != initialHub {
			return RegistryAndTag{}, fmt.Errorf("image %s is not from the same hub as %s", image, environments[0])
		}

		currentTag, err := image.GetTag()
		if err != nil {
			return RegistryAndTag{}, fmt.Errorf("failed to get tag for image %s: %w", image, err)
		}
		if removeDigestSuffix(currentTag) != tag {
			return RegistryAndTag{}, fmt.Errorf("image %s does not have the same tag as %s", image, environments[0])
		}
	}

	return RegistryAndTag{Registry: initialHub, Tag: tag}, nil
}

// RemoveDigestSuffix removes the digest suffix if present (format: @sha256:...)
func removeDigestSuffix(image string) string {
	if idx := strings.Index(image, "@"); idx != -1 {
		return image[:idx]
	}
	return image
}
