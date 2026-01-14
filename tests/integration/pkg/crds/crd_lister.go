package crds

import (
	"context"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"os"
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

// CheckForCRDs checks whether lister CRDs are present on cluster, returns list of CRDs that don't are present / are not present
// in context of shouldHave parameter
func (lister *CRDLister) CheckForCRDs(ctx context.Context, shouldHave bool) ([]string, error) {
	var wrong []string
	for _, kind := range lister.CRDList {
		var u unstructured.Unstructured
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apiextensions.k8s.io",
			Version: "v1",
			Kind:    "CustomResourceDefinition",
		})

		err := lister.k8sClient.Get(ctx, types.NamespacedName{Name: kind}, &u)
		if k8serrors.IsNotFound(err) {
			if shouldHave {
				wrong = append(wrong, kind)
			}
		} else if err != nil {
			return nil, err
		} else if err == nil && !shouldHave {
			wrong = append(wrong, kind)
		}
	}
	return wrong, nil
}
