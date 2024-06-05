package external_installation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ExternalInstall struct {
	cancel context.CancelFunc
	*exec.Cmd
}

func NewExternalInstaller(iopPath, istioVersion string, compatibilityMode bool) (*ExternalInstall, error) {
	var compatibilityParam string
	var err error
	istioInstallPath, ok := os.LookupEnv("ISTIO_INSTALL_BIN_PATH")
	if !ok {
		istioInstallPath = "./istio_install"
	}

	if compatibilityMode {
		compatibilityParam, err = buildCompatibilityParam(istioVersion)
		if err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*6)

	return &ExternalInstall{
		cancel,
		exec.CommandContext(ctx, istioInstallPath, iopPath, compatibilityParam),
	}, nil
}

func (ei *ExternalInstall) Install() error {
	ei.Stdout = os.Stdout
	ei.Stderr = os.Stderr
	defer ei.cancel()
	err := ei.Run()
	if err != nil {
		// We should not return the error of the external process, because it is always "exit status 1" and we do
		// not want to show such an error in the resource status
		return errors.New("istio installation resulted in an error")
	}

	return nil
}

func buildCompatibilityParam(istioVersion string) (string, error) {
	flag := "compatibilityVersion="
	semVerSep := "."
	sp := strings.Split(istioVersion, ".")
	if len(sp) < 3 {
		return "", fmt.Errorf("expected semver Istio version in format X.Y.Z, but got %s", istioVersion)
	}

	major := sp[0]

	tmp, err := strconv.Atoi(sp[1])
	if err != nil {
		return "", err
	}

	minor := strconv.Itoa(tmp - 1)

	compatibilityParam := flag + major + semVerSep + minor

	return compatibilityParam, nil
}
