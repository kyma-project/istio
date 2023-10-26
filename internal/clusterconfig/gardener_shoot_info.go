package clusterconfig

import (
	"context"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConfigMapShootInfoName = "shoot-info"
	ConfigMapShootInfoNS   = "kube-system"
)

func GetDomainName(ctx context.Context, k8sClient client.Client) (string, error) {
	cmShootInfo, err := getShootInfoConfigMap(ctx, k8sClient)
	if err != nil {
		return "", err
	}
	return cmShootInfo.Data["domain"], nil
}

func getProvider(ctx context.Context, k8sClient client.Client) (string, error) {
	cmShootInfo, err := getShootInfoConfigMap(ctx, k8sClient)
	if err != nil {
		return "", err
	}
	return cmShootInfo.Data["provider"], nil
}

func getShootInfoConfigMap(ctx context.Context, k8sClient client.Client) (v1.ConfigMap, error) {
	cm := v1.ConfigMap{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: ConfigMapShootInfoNS, Name: ConfigMapShootInfoName}, &cm)
	if err != nil {
		return v1.ConfigMap{}, err
	}
	return cm, nil
}
