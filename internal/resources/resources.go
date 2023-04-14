package resources

import (
	"context"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type ResourceMeta struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type Resource struct {
	ResourceMeta
	GVK schema.GroupVersionKind
}

type ResourceConfiguration struct {
	GroupVersionKind schema.GroupVersionKind `yaml:"GroupVersionKind"`
	ControlledList   []ResourceMeta          `yaml:"ControlledList"`
}

type resourceFinderConfiguration struct {
	Resources []ResourceConfiguration `yaml:"resources"`
}

type IstioResourcesFinder struct {
	ctx           context.Context
	logger        logr.Logger
	client        client.Client
	configuration resourceFinderConfiguration
}

func NewIstioResourcesFinderFromConfigYaml(ctx context.Context, client client.Client, logger logr.Logger, path string) (*IstioResourcesFinder, error) {
	configYaml, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var finder resourceFinderConfiguration
	err = yaml.Unmarshal(configYaml, &finder)
	if err != nil {
		return nil, err
	}
	return &IstioResourcesFinder{
		ctx:           ctx,
		logger:        logger,
		client:        client,
		configuration: finder,
	}, nil
}

func (i *IstioResourcesFinder) FindUserCreatedIstioResources() ([]Resource, error) {
	var userResources []Resource
	for _, resource := range i.configuration.Resources {
		var u unstructured.UnstructuredList
		u.SetGroupVersionKind(resource.GroupVersionKind)
		err := i.client.List(i.ctx, &u)
		if err != nil {
			return nil, err
		}
		for _, item := range u.Items {
			res := Resource{
				GVK: resource.GroupVersionKind,
				ResourceMeta: ResourceMeta{
					Name:      item.GetName(),
					Namespace: item.GetNamespace(),
				},
			}
			if !contains(resource.ControlledList, res.ResourceMeta) {
				userResources = append(userResources, res)
			}
		}
	}
	return userResources, nil
}
func contains(s []ResourceMeta, e ResourceMeta) bool {
	for _, r := range s {
		if r.Name == e.Name && r.Namespace == e.Namespace {
			return true
		}
	}
	return false
}
