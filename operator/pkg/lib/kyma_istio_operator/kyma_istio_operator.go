package kymaistiooperator

import (
	"reflect"

	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/structpb"
	"istio.io/api/operator/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	kymaIstioOperator "github.com/kyma-project/istio/operator/api/v1alpha1"
)

type KymaIstioOperator struct {
	MeshConfig struct {
		AccessLogEncoding string `json:"accessLogEncoding"`
		AccessLogFile     string `json:"accessLogFile"`
		DefaultConfig     struct {
			GatewayTopology struct {
				NumTrustedProxies uint `json:"numTrustedProxies"`
			} `json:"gatewayTopology"`
		} `json:"defaultConfig"`
	} `json:"meshConfig"`
}


func (k KymaIstioOperator) Merge(previous istioOperator.IstioOperator) (istioOperator.IstioOperator, error) {
	if previous.Spec == nil {
		previous.Spec = &v1alpha1.IstioOperatorSpec{}
	}

	op := kymaIstioOperator.Istio{}

	if previous.Spec.MeshConfig == nil {
		previous.Spec.MeshConfig = &structpb.Struct{}
	}

	previous_map := previous.Spec.MeshConfig.AsMap()

	new_map := structToMap(k.MeshConfig)

	maps.Copy(previous_map, new_map)

	val, err := structpb.NewStruct(previous_map)
	if err != nil {
		return istioOperator.IstioOperator{}, err
	}

	previous.Spec.MeshConfig = val
	return previous, nil
}

func structToMap(item interface{}) map[string]interface{} {
	res := map[string]interface{}{}
	if item == nil {
		return res
	}
	v := reflect.TypeOf(item)
	reflectValue := reflect.ValueOf(item)
	reflectValue = reflect.Indirect(reflectValue)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < v.NumField(); i++ {
		tag := v.Field(i).Tag.Get("json")
		field := reflectValue.Field(i).Interface()
		if tag != "" && tag != "-" {
			if v.Field(i).Type.Kind() == reflect.Struct {
				res[tag] = structToMap(field)
			} else {
				res[tag] = field
			}
		}
	}
	return res
}
