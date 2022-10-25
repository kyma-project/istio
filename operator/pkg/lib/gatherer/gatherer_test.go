package gatherer_test

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	IstioResourceName string = "some-istio"
	IstioNamespace    string = "kyma-system"
	TestLabelKey      string = "test-key"
	TestLabelVal      string = "test-val"
	DefaultNamespace  string = "default"
)

func createClientSet(t *testing.T, objects ...client.Object) client.Client {
	err := v1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()

	return client
}

func Test_GetIstioCR(t *testing.T) {
	kymaSystem := corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: IstioNamespace,
		},
	}

	istio_kymaSystem := v1alpha1.Istio{ObjectMeta: v1.ObjectMeta{
		Name:      IstioResourceName,
		Namespace: IstioNamespace,
		Labels: map[string]string{
			TestLabelKey: TestLabelVal,
		},
	}}

	client := createClientSet(t, &kymaSystem, &istio_kymaSystem)

	istioCr, err := gatherer.GetIstioCR(context.TODO(), client, IstioResourceName, IstioNamespace)

	require.NoError(t, err)
	require.Equal(t, istio_kymaSystem.ObjectMeta.Labels[TestLabelKey], istioCr.ObjectMeta.Labels[TestLabelKey])

	noObjectClient := createClientSet(t, &kymaSystem)
	istioCrNoObject, err := gatherer.GetIstioCR(context.TODO(), noObjectClient, IstioResourceName, IstioNamespace)

	require.Error(t, err)
	require.Nil(t, istioCrNoObject)
}

func Test_ListIstioCR(t *testing.T) {
	kymaSystem := corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: IstioNamespace,
		},
	}

	istio_kymaSystem := v1alpha1.Istio{ObjectMeta: v1.ObjectMeta{
		Name:      IstioResourceName,
		Namespace: IstioNamespace,
		Labels: map[string]string{
			TestLabelKey: TestLabelVal,
		},
	}}

	istio_default := v1alpha1.Istio{ObjectMeta: v1.ObjectMeta{
		Name:      IstioResourceName,
		Namespace: DefaultNamespace,
		Labels: map[string]string{
			TestLabelKey: TestLabelVal,
		},
	}}

	client := createClientSet(t, &kymaSystem, &istio_kymaSystem, &istio_default)

	istioCrNoNamespace, err := gatherer.ListIstioCR(context.TODO(), client)

	require.NoError(t, err)
	require.Len(t, istioCrNoNamespace.Items, 2)

	istioCrKymaSystem, err := gatherer.ListIstioCR(context.TODO(), client, IstioNamespace)

	require.NoError(t, err)
	require.Len(t, istioCrKymaSystem.Items, 1)

	istioBothNamespaces, err := gatherer.ListIstioCR(context.TODO(), client, IstioNamespace, "default")

	require.NoError(t, err)
	require.Len(t, istioBothNamespaces.Items, 2)
}
