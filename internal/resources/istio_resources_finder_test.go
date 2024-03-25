package resources

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var sc *runtime.Scheme

var _ = Describe("Resources", func() {
	sc = runtime.NewScheme()
	Expect(networkingv1alpha3.AddToScheme(sc)).To(Succeed())

	DescribeTable("FindUserCreatedIstioResourcesDescribe", func(ctx context.Context, logger logr.Logger, client client.Client, configuration resourceFinderConfiguration, want []Resource, wantErr bool) {
		i := &IstioResourcesFinder{
			ctx:           ctx,
			logger:        logger,
			client:        client,
			configuration: configuration,
		}
		got, err := i.FindUserCreatedIstioResources()
		Expect(err != nil).To(Equal(wantErr))
		Expect(got).To(BeEquivalentTo(want))
	},
		Entry("should get nothing if there are only default istio resources present", context.TODO(),
			logr.Discard(),
			fake.NewClientBuilder().WithScheme(sc).WithObjects(&networkingv1alpha3.EnvoyFilter{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "istio-system",
					Name:      "stats-filter-1.16",
				},
			}).Build(),
			resourceFinderConfiguration{Resources: []ResourceConfiguration{
				{
					GroupVersionKind: schema.GroupVersionKind{
						Group:   "networking.istio.io",
						Version: "v1alpha3",
						Kind:    "EnvoyFilter",
					},
					ControlledList: []ResourceMeta{
						{
							Name:      "stats-filter-1.16",
							Namespace: "istio-system",
						},
					},
				},
			},
			},
			nil,
			false,
		), Entry("should get resource if there is a customer resource present", context.TODO(),
			logr.Discard(),
			fake.NewClientBuilder().WithScheme(sc).WithObjects(&networkingv1alpha3.EnvoyFilter{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "istio-system",
					Name:      "stats-filter-1.16",
				},
			}, &networkingv1alpha3.EnvoyFilter{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "other-system",
					Name:      "route-to-something",
				},
			}).Build(),
			resourceFinderConfiguration{Resources: []ResourceConfiguration{
				{
					GroupVersionKind: schema.GroupVersionKind{
						Group:   "networking.istio.io",
						Version: "v1alpha3",
						Kind:    "EnvoyFilter",
					},
					ControlledList: []ResourceMeta{
						{
							Name:      "stats-filter-1.16",
							Namespace: "istio-system",
						},
					},
				},
			},
			},
			[]Resource{
				{
					ResourceMeta: ResourceMeta{
						Namespace: "other-system",
						Name:      "route-to-something",
					}, GVK: schema.GroupVersionKind{
						Group:   "networking.istio.io",
						Version: "v1alpha3",
						Kind:    "EnvoyFilter",
					},
				},
			},
			false,
		))
})

var _ = Describe("IstioResourcesFinder", func() {
	It("should succeed when reading controlled resources list configuration", func() {
		_, err := NewIstioResourcesFinder(context.TODO(), nil, logr.Logger{})
		Expect(err).ToNot(HaveOccurred())
	})
})

var _ = DescribeTable("contains", func(a []ResourceMeta, b Resource, should bool) {
	Expect(contains(a, b.ResourceMeta)).To(Equal(should))
},
	Entry("should return true if the array contains the resource", []ResourceMeta{{Name: "test", Namespace: "test-ns"}},
		Resource{ResourceMeta: ResourceMeta{
			Name:      "test",
			Namespace: "test-ns",
		}}, true),
	Entry("should return false if the array doesn't contain the resource", []ResourceMeta{{Name: "test", Namespace: "test-ns"}},
		Resource{ResourceMeta: ResourceMeta{
			Name:      "test",
			Namespace: "test",
		}}, false))

func Test_contains(t *testing.T) {
	type args struct {
		s []ResourceMeta
		e Resource
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				s: []ResourceMeta{
					{Name: "test", Namespace: "test-ns"},
				},
				e: Resource{ResourceMeta: ResourceMeta{
					Name:      "test",
					Namespace: "test-ns",
				}},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := contains(tt.args.s, tt.args.e.ResourceMeta)
			if got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
			if err != nil != tt.wantErr {
				t.Errorf("error happened = %v, wanted %v", err != nil, tt.wantErr)
			}
		})
	}
}
