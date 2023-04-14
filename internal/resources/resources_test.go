package resources

import (
	"context"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func TestIstioResourcesFinder_FindUserCreatedIstioResources(t *testing.T) {
	type fields struct {
		ctx           context.Context
		logger        logr.Logger
		client        client.Client
		configuration resourceFinderConfiguration
	}
	sc := runtime.NewScheme()
	networkingv1alpha3.AddToScheme(sc)
	tests := []struct {
		name    string
		fields  fields
		want    []Resource
		wantErr bool
	}{
		{name: "basic",
			fields: fields{
				ctx:    context.TODO(),
				logger: logr.Discard(),
				client: fake.NewClientBuilder().WithScheme(sc).WithObjects(&networkingv1alpha3.EnvoyFilter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "istio-system",
						Name:      "stats-filter-1.16",
					},
				}).Build(),
				configuration: resourceFinderConfiguration{Resources: []ResourceConfiguration{
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
				}},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IstioResourcesFinder{
				ctx:           tt.fields.ctx,
				logger:        tt.fields.logger,
				client:        tt.fields.client,
				configuration: tt.fields.configuration,
			}
			got, err := i.FindUserCreatedIstioResources()
			if (err != nil) != tt.wantErr {
				t.Errorf("FindUserCreatedIstioResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindUserCreatedIstioResources() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewIstioResourcesFinderFromConfigYaml(t *testing.T) {
	type args struct {
		ctx    context.Context
		client client.Client
		logger logr.Logger
		path   string
	}
	tests := []struct {
		name    string
		args    args
		want    *IstioResourcesFinder
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				ctx:    context.TODO(),
				client: nil,
				logger: logr.Logger{},
				path:   "test_resources_list.yaml",
			},
			want: &IstioResourcesFinder{
				ctx:    context.TODO(),
				logger: logr.Logger{},
				client: nil,
				configuration: resourceFinderConfiguration{
					Resources: []ResourceConfiguration{
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewIstioResourcesFinderFromConfigYaml(tt.args.ctx, tt.args.client, tt.args.logger, tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIstioResourcesFinderFromConfigYaml() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewIstioResourcesFinderFromConfigYaml() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_contains(t *testing.T) {
	type args struct {
		s []ResourceMeta
		e Resource
	}
	tests := []struct {
		name string
		args args
		want bool
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
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.args.s, tt.args.e.ResourceMeta); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
