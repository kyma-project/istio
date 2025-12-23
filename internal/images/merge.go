package images

import (
	"os"

	"github.com/imdario/mergo"
	"sigs.k8s.io/yaml"
)

const pullSecretEnvVar = "SKR_IMG_PULL_SECRET"

// MergeComponentImages merges component-specific image values into the IstioOperator manifest.
// This overrides the values.<component>.image fields with the full image references from environment variables.
// It also sets the global hub and tag to match the registry and tag of the provided images.
func MergeComponentImages(manifest []byte, images Images) ([]byte, error) {
	var templateMap map[string]interface{}
	err := yaml.Unmarshal(manifest, &templateMap)
	if err != nil {
		return nil, err
	}
	istioImagesRegistryAndTag, err := images.GetImageRegistryAndTag()
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

	spec := ensureMap(templateMap, "spec")
	values := ensureMap(spec, "values")

	// Set pilot image: values.pilot.image
	pilot := ensureMap(values, "pilot")
	pilotImage, err := images.Pilot.GetName()
	if err != nil {
		return nil, err
	}
	pilot["image"] = pilotImage

	// Set CNI image: values.cni.image
	cni := ensureMap(values, "cni")
	installCNI, err := images.InstallCNI.GetName()
	if err != nil {
		return nil, err
	}
	cni["image"] = installCNI

	// Set proxy image: values.global.proxy.image
	global := ensureMap(values, "global")
	proxy := ensureMap(global, "proxy")
	proxyV2, err := images.ProxyV2.GetName()
	if err != nil {
		return nil, err
	}
	proxy["image"] = proxyV2

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
