package crds

import (
	"context"
	"errors"
	"fmt"
	"os"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type CRDLister struct {
	k8sClient client.Client
	CRDList   []string
}

func NewCRDListerFromFile(k8sClient client.Client, path string) (*CRDLister, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var lister CRDLister
	err = yaml.Unmarshal(fileContent, &lister)
	if err != nil {
		return nil, err
	}
	if len(lister.CRDList) == 0 {
		return nil, errors.New("CRDList is empty")
	}
	lister.k8sClient = k8sClient

	return &lister, nil
}

func (lister *CRDLister) EnsureCRDsArePresent(ctx context.Context) error {
	var missing []string
	for _, kind := range lister.CRDList {
		var u unstructured.Unstructured
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apiextensions.k8s.io",
			Version: "v1",
			Kind:    "CustomResourceDefinition",
		})

		err := lister.k8sClient.Get(ctx, types.NamespacedName{Name: kind}, &u)

		if err != nil {
			if k8serrors.IsNotFound(err) {
				missing = append(missing, kind)
			} else {
				return fmt.Errorf("error getting CRD %s: %v", kind, err)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("the following CRDs are missing: %v", missing)
	}

	return nil
}

func (lister *CRDLister) EnsureCRDsAreNotPresent(ctx context.Context) error {
	var exists []string
	for _, kind := range lister.CRDList {
		var u unstructured.Unstructured
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apiextensions.k8s.io",
			Version: "v1",
			Kind:    "CustomResourceDefinition",
		})

		err := lister.k8sClient.Get(ctx, types.NamespacedName{Name: kind}, &u)

		if !k8serrors.IsNotFound(err) {
			return fmt.Errorf("error getting CRD %s: %v", kind, err)
		}

		if err == nil {
			exists = append(exists, kind)
		}

	}

	if len(exists) > 0 {
		return fmt.Errorf("the following CRDs are present but shouldn't: %v", exists)
	}

	return nil
}
