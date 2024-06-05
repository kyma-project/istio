package istio

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

var errCannotParseSemver = errors.New("cannot parse semver, invalid format")

type externalInstaller struct {
	cancel context.CancelFunc
	*exec.Cmd
}

func newExternalInstaller(iopPath, istioVersion string, compatibilityMode bool) (*externalInstaller, error) {
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

	return &externalInstaller{
		cancel,
		exec.CommandContext(ctx, istioInstallPath, iopPath, compatibilityParam),
	}, nil
}

func (ei *externalInstaller) Install() error {
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
	sp := strings.Split(istioVersion, ".")
	if len(sp) < 3 {
		return "", errCannotParseSemver
	}

	majorVersion := sp[0]
	minorVersion, err := decreaseOneMinorVersion(sp[1])
	if err != nil {
		return "", err
	}

	compatibilityParam := fmt.Sprintf("compatibilityVersion=%s.%s", majorVersion, minorVersion)
	return compatibilityParam, nil
}

func decreaseOneMinorVersion(minor string) (string, error) {
	tmp, err := strconv.Atoi(minor)
	if err != nil {
		return "", err
	}

	minorBackOne := strconv.Itoa(tmp - 1)
	return minorBackOne, nil
}
