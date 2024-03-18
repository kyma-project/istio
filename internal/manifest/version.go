package manifest

import (
	"fmt"

	"github.com/coreos/go-semver/semver"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
)

// variable is set to the correct version by the Dockerfile during build time.
var version = "dev"

func GetModuleVersion() string {
	return version
}

func GetIstioVersion(merger Merger) (string, string, error) {
	iop, err := merger.GetIstioOperator(clusterconfig.Production)
	if err != nil {
		return "", "", err
	}

	v, err := semver.NewVersion(iop.Spec.Tag.GetStringValue())
	if err != nil {
		return "", "", err
	}

	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch), string(v.PreRelease), nil
}
