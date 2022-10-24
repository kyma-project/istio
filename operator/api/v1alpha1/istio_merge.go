package v1alpha1

import (
	"reflect"
	"strings"

	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/structpb"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
)

// Merge implements builder.Mergeable
func (i Istio) Merge(operator istioOperator.IstioOperator) (istioOperator.IstioOperator, error) {
	prevMeshConfigMap := operator.Spec.MeshConfig.AsMap()
	opMeshConfigMap := structToMap(i.Spec.Controlplane.MeshConfig)

	maps.Copy(prevMeshConfigMap, opMeshConfigMap)

	newMeshConfig, err := structpb.NewStruct(prevMeshConfigMap)
	if err != nil {
		return istioOperator.IstioOperator{}, err
	}

	operator.Spec.MeshConfig = newMeshConfig

	return operator, nil
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
			tag = strings.TrimSuffix(tag, ",omitempty")
			if v.Field(i).Type.Kind() == reflect.Struct {
				res[tag] = structToMap(field)
			} else {
				res[tag] = field
			}
		}
	}
	return res
}
