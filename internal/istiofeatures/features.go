package istiofeatures

import (
	"context"
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	featuresConfigMapName      = "istio-features"
	featuresConfigMapNamespace = "kyma-system"

	configMapKey = "features"
)

type IstioFeatures struct {
	DisableCni bool `json:"disableCni"`
}

func Get(ctx context.Context, k8sClient client.Client) (IstioFeatures, error) {
	var IstioFeaturesConfigMap corev1.ConfigMap
	err := k8sClient.Get(ctx, types.NamespacedName{Name: featuresConfigMapName, Namespace: featuresConfigMapNamespace}, &IstioFeaturesConfigMap)
	if err != nil {
		return IstioFeatures{}, err
	}

	featuresData, ok := IstioFeaturesConfigMap.Data[configMapKey]
	if !ok {
		return IstioFeatures{}, nil
	}

	var features IstioFeatures
	err = json.Unmarshal([]byte(featuresData), &features)
	return features, err
}
