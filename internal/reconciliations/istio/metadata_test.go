package istio

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func Test_updateResourcesMetadataForSelector(t *testing.T) {
	testPod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"install.operator.istio.io/owning-resource": "installed-state-default-operator",
			},
		}}
	client := fake.NewFakeClient(&testPod)
	err := updateResourcesMetadataForSelector(context.Background(), client)
	if err != nil {
		t.Error(err)
	}
	got := v1.Pod{}
	wanted := map[string]string{
		"install.operator.istio.io/owning-resource": "installed-state-default-operator",
		"kyma-project.io/module":                    "istio",
	}
	err = client.Get(context.TODO(), types.NamespacedName{Name: "test-pod", Namespace: "default"}, &got)
	if err != nil {
		t.Error(err)
	}
	if len(got.GetLabels()) == 2 && !reflect.DeepEqual(got.Labels, wanted) {
		t.Errorf("Module label has not been set correctly: got %v, wanted %v\n", got.Labels, wanted)
	}
}
