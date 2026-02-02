package images

import (
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
)

const kymaFipsModeEnabledEnv = "KYMA_FIPS_MODE_ENABLED"

type RegistryAndTag struct {
	Registry string
	Tag      string
}

type images struct {
	Pilot      string `env:"pilot,notEmpty"`
	InstallCNI string `env:"install-cni,notEmpty"`
	ProxyV2    string `env:"proxyv2,notEmpty"`
	Ztunnel    string `env:"ztunnel"`
}

type imagesFips struct {
	Pilot      string `env:"pilot-fips,notEmpty"`
	InstallCNI string `env:"install-cni-fips,notEmpty"`
	ProxyV2    string `env:"proxyv2-fips,notEmpty"`
	Ztunnel    string `env:"ztunnel-fips"`
}

func GetImages() (*Images, error) {
	envImages, err := parsImagesFromEnvs()
	if err != nil {
		return nil, err
	}
	i := &Images{}

	parsedImage, err := parseImage(envImages.Pilot)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pilot image: %w", err)
	}
	i.Pilot = parsedImage

	parsedImage, err = parseImage(envImages.InstallCNI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse install-cni image: %w", err)
	}
	i.InstallCNI = parsedImage

	parsedImage, err = parseImage(envImages.ProxyV2)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxyv2 image: %w", err)
	}
	i.ProxyV2 = parsedImage

	if envImages.Ztunnel != "" {
		parsedImage, err = parseImage(envImages.Ztunnel)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ztunnel image: %w", err)
		}
		i.Ztunnel = parsedImage
	}

	if err := i.getImageRegistryAndTag(); err != nil {
		return nil, err
	}

	return i, nil
}

func parsImagesFromEnvs() (*images, error) {
	var err error
	var environments images
	var fipsEnvironments imagesFips

	kymaFipsModeEnabled := os.Getenv(kymaFipsModeEnabledEnv)
	if kymaFipsModeEnabled == "true" {
		fipsEnvironments, err = env.ParseAs[imagesFips]()
		if err != nil {
			return nil, fmt.Errorf("missing required FIPS environment variables %w", err)
		}
		return (*images)(&fipsEnvironments), nil
	}
	environments, err = env.ParseAs[images]()
	if err != nil {
		return nil, fmt.Errorf("missing required environment variables %w", err)
	}
	return &environments, nil

}

// parseImage parses an OCI image reference into Image to have parsed and available to get: registry, name, and tag.
// Supports formats like:
//   - registry/namespace/image:tag
//   - registry:port/namespace/image:tag
//   - registry/namespace/image:tag@sha256:digest
//   - registry:port/namespace/image:tag@sha256:digest
func parseImage(image string) (Image, error) {
	if image == "" {
		return Image{}, fmt.Errorf("image can not be empty")
	}

	// Find the last "/" to separate registry from image name
	lastSlash := strings.LastIndex(image, "/")
	if lastSlash == -1 {
		return Image{}, fmt.Errorf("image %s does not contain a valid format", image)
	}

	registry := image[:lastSlash]
	namePart := image[lastSlash+1:]

	// Find the tag separator ":" in the image name portion
	colonIdx := strings.Index(namePart, ":")
	if colonIdx == -1 {
		return Image{}, fmt.Errorf("image %s does not contain a valid tag", image)
	}
	return Image{
		Registry: registry,
		Name:     namePart[:colonIdx],
		Tag:      namePart[colonIdx+1:],
	}, nil
}

func (e *Images) getImageRegistryAndTag() error {
	environments := []Image{e.Pilot, e.InstallCNI, e.ProxyV2}
	if e.Ztunnel.Name != "" {
		environments = append(environments, e.Ztunnel)
	}

	initialHub := environments[0].Registry
	initialTag := environments[0].Tag
	tag := removeDigestSuffix(initialTag)

	// Ensure that all required images are from the same hub and have the same version tag
	for _, image := range environments {
		currentHub := image.Registry
		if currentHub != initialHub {
			return fmt.Errorf("image %s is not from the same hub as %s", image, environments[0])
		}

		if removeDigestSuffix(image.Tag) != tag {
			return fmt.Errorf("image %s does not have the same tag as %s", image, environments[0])
		}
	}

	e.Registry = initialHub
	e.Tag = tag
	return nil
}
