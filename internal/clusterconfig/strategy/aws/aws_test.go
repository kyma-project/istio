package aws_test

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig/strategy/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newFakeClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func elbDeprecatedCM() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{Name: "elb-deprecated", Namespace: "istio-system"},
	}
}

func ingressGatewaySvc(annotations map[string]string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:        "istio-ingressgateway",
			Namespace:   "istio-system",
			Annotations: annotations,
		},
	}
}

func TestNewStrategy_LBSelection(t *testing.T) {
	tests := []struct {
		name             string
		objs             []client.Object
		dualStackEnabled bool
		wantAnnots       map[string]string
	}{
		{
			name:             "no elb-deprecated CM -> NLB IPv4",
			objs:             nil,
			dualStackEnabled: false,
			wantAnnots: map[string]string{
				aws.LBTypeAnnotation:        aws.NLBType,
				aws.SchemeAnnotation:        aws.InternetFacingScheme,
				aws.NlbTargetTypeAnnotation: aws.NlbTargetTypeInstance,
			},
		},
		{
			name:             "no elb-deprecated CM + dualStack -> NLB (static annotation set; dualstack not yet wired)",
			objs:             nil,
			dualStackEnabled: true,
			wantAnnots: map[string]string{
				aws.LBTypeAnnotation:        aws.NLBType,
				aws.SchemeAnnotation:        aws.InternetFacingScheme,
				aws.NlbTargetTypeAnnotation: aws.NlbTargetTypeInstance,
			},
		},
		{
			name:             "elb-deprecated CM present, no service -> ELB (no annotations)",
			objs:             []client.Object{elbDeprecatedCM()},
			dualStackEnabled: false,
			wantAnnots:       nil,
		},
		{
			name: "elb-deprecated CM present + service already NLB -> NLB IPv4",
			objs: []client.Object{
				elbDeprecatedCM(),
				ingressGatewaySvc(map[string]string{
					aws.LBTypeAnnotation: aws.NLBType,
				}),
			},
			dualStackEnabled: false,
			wantAnnots: map[string]string{
				aws.LBTypeAnnotation:        aws.NLBType,
				aws.SchemeAnnotation:        aws.InternetFacingScheme,
				aws.NlbTargetTypeAnnotation: aws.NlbTargetTypeInstance,
			},
		},
		{
			name: "elb-deprecated CM present + service with non-NLB annotation -> ELB",
			objs: []client.Object{
				elbDeprecatedCM(),
				ingressGatewaySvc(map[string]string{
					aws.LBTypeAnnotation: "classic",
				}),
			},
			dualStackEnabled: false,
			wantAnnots:       nil,
		},
		{
			name: "elb-deprecated CM present + service without LB annotation -> ELB",
			objs: []client.Object{
				elbDeprecatedCM(),
				ingressGatewaySvc(nil),
			},
			dualStackEnabled: false,
			wantAnnots:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newFakeClient(t, tt.objs...)

			s, err := aws.NewStrategy(context.Background(), c, tt.dualStackEnabled)
			require.NoError(t, err)
			require.NotNil(t, s)
			require.NotNil(t, s.LB)

			annots := s.GetLBAnnotations()
			assert.Equal(t, tt.wantAnnots, annots)
		})
	}
}

func TestLB_RequiresProxyProtocolEnvoyFilter(t *testing.T) {
	tests := []struct {
		name             string
		objs             []client.Object
		dualStackEnabled bool
		want             bool
	}{
		{
			name:             "NLB IPv4 does not require proxy protocol envoy filter",
			objs:             nil,
			dualStackEnabled: false,
			want:             false,
		},
		{
			name:             "NLB DualStack requires proxy protocol envoy filter",
			objs:             nil,
			dualStackEnabled: true,
			want:             true,
		},
		{
			name:             "ELB requires proxy protocol envoy filter",
			objs:             []client.Object{elbDeprecatedCM()},
			dualStackEnabled: false,
			want:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newFakeClient(t, tt.objs...)
			s, err := aws.NewStrategy(context.Background(), c, tt.dualStackEnabled)
			require.NoError(t, err)
			assert.Equal(t, tt.want, s.RequiresProxyProtocolEnvoyFilter())
		})
	}
}

func TestNewStrategy_NoCNI(t *testing.T) {
	c := newFakeClient(t)
	s, err := aws.NewStrategy(context.Background(), c, false)
	require.NoError(t, err)
	assert.Nil(t, s.CNI)
}
