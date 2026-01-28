package crds

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var crdMissingError = errors.New("CRDs missing")

func AssertIstioCRDsPresent(ctx context.Context, c client.Client) error {
	_, filename, _, _ := runtime.Caller(0)
	packageDir := filepath.Dir(filename)

	l, err := NewCRDListerFromFile(c, packageDir+"/istio_crd_list.yaml")
	if err != nil {
		return err
	}

	err = l.checkCrdsExist(ctx)
	if err != nil {
		return err
	}

	return nil
}

func AssertIstioCRDsNotPresent(ctx context.Context, c client.Client) error {
	_, filename, _, _ := runtime.Caller(0)
	packageDir := filepath.Dir(filename)

	l, err := NewCRDListerFromFile(c, packageDir+"/istio_crd_list.yaml")
	if err != nil {
		return err
	}

	err = l.checkCrdsExist(ctx)

	if err != nil {
		if errors.Is(err, crdMissingError) {
			return nil
		}
		return err
	}

	return errors.New("expected Istio CRDs to not to be present")
}
