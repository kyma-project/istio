package gatherer_test

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	"github.com/masterminds/semver"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	IstioResourceName    string = "some-istio"
	IstioCRNamespace     string = "kyma-system"
	IstioSystemNamespace string = "istio-system"
	TestLabelKey         string = "test-key"
	TestLabelVal         string = "test-val"
	DefaultNamespace     string = "default"
)

func createClientSet(t *testing.T, objects ...client.Object) client.Client {
	err := v1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	err = appsv1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()

	return client
}

func Test_GetIstioCR(t *testing.T) {
	kymaSystem := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: IstioCRNamespace,
		},
	}

	istio_kymaSystem := v1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
		Name:      IstioResourceName,
		Namespace: IstioCRNamespace,
		Labels: map[string]string{
			TestLabelKey: TestLabelVal,
		},
	}}

	client := createClientSet(t, &kymaSystem, &istio_kymaSystem)

	istioCr, err := gatherer.GetIstioCR(context.TODO(), client, IstioResourceName, IstioCRNamespace)

	require.NoError(t, err)
	require.Equal(t, istio_kymaSystem.ObjectMeta.Labels[TestLabelKey], istioCr.ObjectMeta.Labels[TestLabelKey])

	noObjectClient := createClientSet(t, &kymaSystem)
	istioCrNoObject, err := gatherer.GetIstioCR(context.TODO(), noObjectClient, IstioResourceName, IstioCRNamespace)

	require.Error(t, err)
	require.Nil(t, istioCrNoObject)
}

func Test_ListIstioCR(t *testing.T) {
	kymaSystem := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: IstioCRNamespace,
		},
	}

	istio_kymaSystem := v1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
		Name:      IstioResourceName,
		Namespace: IstioCRNamespace,
		Labels: map[string]string{
			TestLabelKey: TestLabelVal,
		},
	}}

	istio_default := v1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
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

	istioCrKymaSystem, err := gatherer.ListIstioCR(context.TODO(), client, IstioCRNamespace)

	require.NoError(t, err)
	require.Len(t, istioCrKymaSystem.Items, 1)

	istioBothNamespaces, err := gatherer.ListIstioCR(context.TODO(), client, IstioCRNamespace, "default")

	require.NoError(t, err)
	require.Len(t, istioBothNamespaces.Items, 2)
}

func Test_ListInstalledIstioRevisions(t *testing.T) {
	t.Run("Should list all istio versions with revisions", func(t *testing.T) {

		istioSystem := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: IstioSystemNamespace,
			},
		}

		istiod_defaultRevision := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "istiod",
				Labels: map[string]string{
					"app":          "istiod",
					"istio.io/rev": "default",
					"operator.istio.io/version": "1.16.1",
				},
			},
		}

		istiod_otherRevision := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "istiod-stable",
				Labels: map[string]string{
					"app":          "istiod",
					"istio.io/rev": "stable",
					"operator.istio.io/version": "1.15.4",
				},
			},
		}

		client := createClientSet(t, &istioSystem, &istiod_defaultRevision, &istiod_otherRevision)

		istioVersions, err := gatherer.ListInstalledIstioRevisions(context.TODO(), client)

		require.NoError(t, err)

		require.Contains(t, istioVersions, "default")
		require.Equal(t, istioVersions["default"], semver.MustParse("1.16.1"))

		require.Contains(t, istioVersions, "stable")
		require.Equal(t, istioVersions["stable"], semver.MustParse("1.15.4"))
	})
}
