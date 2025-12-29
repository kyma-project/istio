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

func (i Image) GetHub() (string, error) {
	if i == "" {
		return "", fmt.Errorf("image can not be empty")
	}

	parts := strings.Split(string(i), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("image %s does not contain a valid hub URL", i)
	}

	return strings.Join(parts[:len(parts)-1], "/"), nil
}

func (i Image) GetTag() (string, error) {
	if i == "" {
		return "", fmt.Errorf("image can not be empty")
	}

	parts := strings.Split(string(i), ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("image %s does not contain a valid tag", i)
	}

	return parts[len(parts)-1], nil
}

func (i Image) GetName() (string, error) {
	if i == "" {
		return "", fmt.Errorf("image can not be empty")
	}

	parts := strings.Split(string(i), ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("image %s does not contain a valid image name", i)
	}
	parts = strings.Split(parts[0], "/")
	return parts[len(parts)-1], nil
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
		if currentTag != initialTag {
			return RegistryAndTag{}, fmt.Errorf("image %s does not have the same tag as %s", image, environments[0])
		}
	}

	return RegistryAndTag{Registry: initialHub, Tag: initialTag}, nil
}
