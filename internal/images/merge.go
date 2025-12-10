package images

import (
	"os"

	"github.com/imdario/mergo"
	"sigs.k8s.io/yaml"
)

const pullSecretEnvVar = "SKR_IMG_PULL_SECRET"

// MergeRegistryAndTagConfiguration merges the Istio hub and tag configuration to the provided manifest.
func MergeRegistryAndTagConfiguration(manifest []byte, istioImagesRegistryAndTag RegistryAndTag) ([]byte, error) {
	var templateMap map[string]interface{}
	err := yaml.Unmarshal(manifest, &templateMap)
	if err != nil {
		return nil, err
	}

	err = mergo.Merge(&templateMap, map[string]interface{}{
		"spec": map[string]interface{}{
			"hub": istioImagesRegistryAndTag.Registry,
			"tag": istioImagesRegistryAndTag.Tag,
		},
	}, mergo.WithOverride)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(templateMap)
}

func MergePullSecretEnv(manifest []byte) ([]byte, error) {
	secret, pullSecretEnvExists := os.LookupEnv(pullSecretEnvVar)
	if !pullSecretEnvExists {
		return manifest, nil
	}

	var templateMap map[string]interface{}
	_ = yaml.Unmarshal(manifest, &templateMap)

	spec := ensureMap(templateMap, "spec")
	values := ensureMap(spec, "values")
	global := ensureMap(values, "global")

	ips, ok := global["imagePullSecrets"].([]interface{})
	if !ok {
		ips = []interface{}{}
	}

	secretName := secret
	already := false
	for _, v := range ips {
		if v == secretName {
			already = true
			break
		}
	}
	if !already {
		ips = append(ips, secretName)
	}

	global["imagePullSecrets"] = ips

	out, err := yaml.Marshal(templateMap)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func ensureMap(m map[string]interface{}, key string) map[string]interface{} {
	v, ok := m[key]
	if !ok {
		nm := map[string]interface{}{}
		m[key] = nm
		return nm
	}
	nm, ok := v.(map[string]interface{})
	if !ok {
		nm = map[string]interface{}{}
		m[key] = nm
	}
	return nm
}
