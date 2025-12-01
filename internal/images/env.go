package images

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Image string

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

type Images struct {
	Pilot      Image `env:"pilot,notEmpty"`
	InstallCNI Image `env:"install-cni,notEmpty"`
	ProxyV2    Image `env:"proxyv2,notEmpty"`
	Ztunnel    Image `env:"ztunnel,notEmpty"`
}

func GetImages() (*Images, error) {
	environments, err := env.ParseAs[Images]()
	if err != nil {
		return nil, fmt.Errorf("missing required environment variables %w", err)
	}

	return &environments, nil
}

func (e *Images) GetHub() (string, error) {
	environments := []Image{e.Pilot, e.InstallCNI, e.ProxyV2}

	initialHub, err := environments[0].GetHub()
	if err != nil {
		return "", fmt.Errorf("failed to get hub for image %s: %w", environments[0], err)
	}
	// Ensure that all required images are from the same hub
	for _, image := range environments {
		currentHub, err := image.GetHub()
		if err != nil {
			return "", fmt.Errorf("failed to get hub for image %s: %w", image, err)
		}

		if currentHub != initialHub {
			return "", fmt.Errorf("image %s is not from the same hub as %s", image, initialHub)
		}
	}
	return initialHub, nil
}
