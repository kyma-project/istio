package images

import (
	"github.com/imdario/mergo"
	"sigs.k8s.io/yaml"
)

// MergeHubConfiguration merges the Istio hub configuration to the provided manifest.
func MergeHubConfiguration(manifest []byte, istioImagesHub string) ([]byte, error) {
	var templateMap map[string]interface{}
	err := yaml.Unmarshal(manifest, &templateMap)
	if err != nil {
		return nil, err
	}

	err = mergo.Merge(&templateMap, map[string]interface{}{
		"spec": map[string]interface{}{
			"hub": istioImagesHub,
		},
	}, mergo.WithOverride)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(templateMap)
}
