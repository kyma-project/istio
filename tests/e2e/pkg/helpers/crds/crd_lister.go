package crds

import (
	"context"
	"errors"
	"fmt"
	"os"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
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
	lister.k8sClient = k8sClient

	return &lister, nil
}

func (lister *CRDLister) checkCrdsExist(ctx context.Context) error {
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
			if !k8sErrors.IsNotFound(err) {
				return err
			}
			missing = append(missing, kind)
		}

	}

	if len(missing) > 0 {
		err := fmt.Errorf("missing CRDs: %v", missing)
		return errors.Join(crdMissingError, err)
	}

	return nil
}
