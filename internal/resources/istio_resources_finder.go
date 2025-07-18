package resources

import (
	"context"
	"fmt"
	"regexp"

	_ "embed"

	"github.com/kyma-project/istio/operator/pkg/labels"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

//go:embed controlled_resources_list.yaml
var controlledResourcesList []byte

type ResourceMeta struct {
	Name      string
	Namespace string
}

type Resource struct {
	ResourceMeta
	GVK schema.GroupVersionKind
}

type ResourceConfiguration struct {
	GroupVersionKind schema.GroupVersionKind
	ControlledList   []ResourceMeta
}

type resourceFinderConfiguration struct {
	Resources []ResourceConfiguration
}

type IstioResourcesFinder struct {
	ctx           context.Context
	logger        logr.Logger
	client        client.Client
	configuration resourceFinderConfiguration
}

var noMatchesForKind = regexp.MustCompile("no matches for kind")
var couldNotFindReqResource = regexp.MustCompile("could not find the requested resource")

func NewIstioResourcesFinder(ctx context.Context, client client.Client, logger logr.Logger) (*IstioResourcesFinder, error) {
	var finderConfiguration resourceFinderConfiguration
	err := yaml.Unmarshal(controlledResourcesList, &finderConfiguration)
	if err != nil {
		return nil, err
	}

	for _, resource := range finderConfiguration.Resources {
		for _, meta := range resource.ControlledList {
			_, compileErr := regexp.Compile(meta.Name)
			if compileErr != nil {
				return nil, fmt.Errorf("configuration yaml regex check failed for \"%s\": %w", meta.Name, compileErr)
			}

			_, compileErr = regexp.Compile(meta.Namespace)
			if compileErr != nil {
				return nil, fmt.Errorf("configuration yaml regex check failed for \"%s\": %w", meta.Namespace, compileErr)
			}
		}
	}

	return &IstioResourcesFinder{
		ctx:           ctx,
		logger:        logger,
		client:        client,
		configuration: finderConfiguration,
	}, nil
}

func (i *IstioResourcesFinder) FindUserCreatedIstioResources() ([]Resource, error) {
	var userResources []Resource
	for _, resource := range i.configuration.Resources {
		var u unstructured.UnstructuredList
		u.SetGroupVersionKind(resource.GroupVersionKind)
		err := i.client.List(i.ctx, &u)
		if err != nil {
			if errors.IsNotFound(err) || noMatchesForKind.MatchString(err.Error()) || couldNotFindReqResource.MatchString(err.Error()) {
				continue
			}
			return nil, err
		}
		for _, item := range u.Items {
			if labels.HasModuleLabels(item) {
				continue
			}
			res := Resource{
				GVK: resource.GroupVersionKind,
				ResourceMeta: ResourceMeta{
					Name:      item.GetName(),
					Namespace: item.GetNamespace(),
				},
			}
			managed, resourceErr := contains(resource.ControlledList, res.ResourceMeta)
			if resourceErr != nil {
				return nil, resourceErr
			}
			if !managed {
				userResources = append(userResources, res)
			}
		}
	}
	return userResources, nil
}
func contains(s []ResourceMeta, e ResourceMeta) (bool, error) {
	for _, r := range s {
		matchName, err := regexp.MatchString(r.Name, e.Name)
		if err != nil {
			return false, err
		}
		matchNamespace, err := regexp.MatchString(r.Namespace, e.Namespace)
		if err != nil {
			return false, err
		}
		if matchNamespace && matchName {
			return true, nil
		}
	}
	return false, nil
}
