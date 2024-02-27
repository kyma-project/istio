package steps

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"strings"

	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/controllers"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	securityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type godogResourceMapping int

func (k godogResourceMapping) String() string {
	switch k {
	case DaemonSet:
		return "DaemonSet"
	case Deployment:
		return "Deployment"
	case IstioCR:
		return "Istio CR"
	case DestinationRule:
		return "DestinationRule"
	case Namespace:
		return "Namespace"
	case Gateway:
		return "Gateway"
	case EnvoyFilter:
		return "EnvoyFilter"
	case PeerAuthentication:
		return "PeerAuthentication"
	case VirtualService:
		return "VirtualService"
	case ConfigMap:
		return "ConfigMap"
	case IstioOperator:
		return "istiooperator"
	}
	panic(fmt.Errorf("%#v has unimplemented String() method", k))
}

const (
	DaemonSet godogResourceMapping = iota
	Deployment
	IstioCR
	DestinationRule
	Namespace
	Gateway
	EnvoyFilter
	PeerAuthentication
	VirtualService
	ConfigMap
	IstioOperator
)

func ResourceIsReady(ctx context.Context, kind, name, namespace string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	return retry.Do(func() error {
		var object client.Object

		switch kind {
		case Deployment.String():
			object = &v1.Deployment{}
		case DaemonSet.String():
			object = &v1.DaemonSet{}
		default:
			return godog.ErrUndefined
		}

		err := k8sClient.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, object)
		if err != nil {
			return err
		}

		switch kind {
		case Deployment.String():
			if object.(*v1.Deployment).Status.Replicas != object.(*v1.Deployment).Status.ReadyReplicas {
				return fmt.Errorf("%s %s/%s is not ready",
					kind, namespace, name)
			}
		case DaemonSet.String():
			if object.(*v1.DaemonSet).Status.NumberReady != object.(*v1.DaemonSet).Status.DesiredNumberScheduled {
				return fmt.Errorf("%s %s/%s is not ready",
					kind, namespace, name)
			}
		default:
			return godog.ErrUndefined
		}

		return nil
	}, testcontext.GetRetryOpts()...)
}

func ResourceIsPresent(ctx context.Context, kind, name, namespace, present string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	return retry.Do(func() error {
		var object client.Object

		switch kind {
		case Gateway.String():
			object = &networkingv1alpha3.Gateway{}
		case EnvoyFilter.String():
			object = &networkingv1alpha3.EnvoyFilter{}
		case PeerAuthentication.String():
			object = &securityv1beta1.PeerAuthentication{}
		case VirtualService.String():
			object = &networkingv1beta1.VirtualService{}
		case ConfigMap.String():
			object = &corev1.ConfigMap{}
		case IstioOperator.String():
			object = &istioOperator.IstioOperator{}
		default:
			return godog.ErrUndefined
		}

		err := k8sClient.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, object)
		if err != nil {
			if present == "not present" && k8serrors.IsNotFound(err) {
				return nil
			}
			return err
		}

		if present == "not present" {
			return fmt.Errorf("%s/%s in ns %s should have not been present", kind, name, namespace)
		}
		return nil
	}, testcontext.GetRetryOpts()...)
}

func EvaluatedClusterSizeIs(ctx context.Context, size string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	clusterSize, err := clusterconfig.EvaluateClusterSize(ctx, k8sClient)
	if err != nil {
		return err
	}

	if clusterSize.String() != size {
		return fmt.Errorf("evaluated cluster size %s is not %s", clusterSize.String(), size)
	}

	return nil
}

func NamespaceIsCreated(ctx context.Context, name string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	err = retry.Do(func() error {
		err := k8sClient.Create(ctx, &ns)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, &ns)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

func NamespaceIsPresent(ctx context.Context, name, shouldBePresent string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	var ns corev1.Namespace
	return retry.Do(func() error {
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name}, &ns)
		if shouldBePresent != "present" {
			if !k8serrors.IsNotFound(err) {
				return fmt.Errorf("namespace %s is present but shouldn't", name)
			}
			return nil
		}
		return err
	}, testcontext.GetRetryOpts()...)
}

func NamespaceHasLabelAndAnnotation(ctx context.Context, name, label, annotation string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	var ns corev1.Namespace
	return retry.Do(func() error {
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name}, &ns)
		if _, ok := ns.Labels[label]; !ok {
			return fmt.Errorf("namespace %s does not contain %s label", name, label)
		}
		if _, ok := ns.Annotations[annotation]; !ok {
			return fmt.Errorf("namespace %s does not contain %s annotation", name, annotation)
		}
		return err
	}, testcontext.GetRetryOpts()...)
}

func ClusterResourceIsDeleted(ctx context.Context, kind, name string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	switch kind {
	case Namespace.String():
		return retry.Do(func() error {
			err := k8sClient.Delete(context.TODO(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}})
			return err
		})
	default:
		return fmt.Errorf("can't delete resource for undefined kind %s", kind)
	}
}

func ResourceInNamespaceIsDeleted(ctx context.Context, kind, name, namespace string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	switch kind {
	case IstioCR.String():
		return retry.Do(func() error {
			var istioCr istioCR.Istio
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &istioCr)
			if err != nil {
				return err
			}

			return k8sClient.Delete(context.TODO(), &istioCr)
		})
	case DestinationRule.String():
		return retry.Do(func() error {
			var dr networkingv1beta1.DestinationRule
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &dr)
			if err != nil {
				return err
			}

			return k8sClient.Delete(context.TODO(), &dr)
		})
	case Deployment.String():
		return retry.Do(func() error {
			var dep v1.Deployment
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &dep)
			if err != nil {
				return err
			}

			return k8sClient.Delete(context.TODO(), &dep)
		})
	case DaemonSet.String():
		return retry.Do(func() error {
			var r v1.DaemonSet
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &r)
			if err != nil {
				return err
			}

			return k8sClient.Delete(context.TODO(), &r)
		})
	case Gateway.String():
		return retry.Do(func() error {
			var r networkingv1beta1.Gateway
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &r)
			if err != nil {
				return err
			}

			return k8sClient.Delete(context.TODO(), &r)
		})
	case EnvoyFilter.String():
		return retry.Do(func() error {
			var r networkingv1alpha3.EnvoyFilter
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &r)
			if err != nil {
				return err
			}

			return k8sClient.Delete(context.TODO(), &r)
		})
	case PeerAuthentication.String():
		return retry.Do(func() error {
			var r securityv1beta1.PeerAuthentication
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &r)
			if err != nil {
				return err
			}

			return k8sClient.Delete(context.TODO(), &r)
		})
	case VirtualService.String():
		return retry.Do(func() error {
			var r networkingv1beta1.VirtualService
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &r)
			if err != nil {
				return err
			}

			return k8sClient.Delete(context.TODO(), &r)
		})
	case ConfigMap.String():
		return retry.Do(func() error {
			var r corev1.ConfigMap
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &r)
			if err != nil {
				return err
			}

			return k8sClient.Delete(context.TODO(), &r)
		})
	case IstioOperator.String():
		return retry.Do(func() error {
			var r istioOperator.IstioOperator
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &r)
			if err != nil {
				return err
			}

			return k8sClient.Delete(context.TODO(), &r)
		})
	default:
		return fmt.Errorf("can't delete resource for undefined kind %s", kind)
	}
}

func ResourceNotPresent(ctx context.Context, kind string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	return retry.Do(func() error {
		switch kind {
		case IstioCR.String():
			var istioList istioCR.IstioList
			err := k8sClient.List(context.TODO(), &istioList)
			if err != nil {
				return err
			}
			if len(istioList.Items) > 0 {
				return fmt.Errorf("there are %d %s present but shouldn't", len(istioList.Items), kind)
			}

		case DestinationRule.String():
			var drList networkingv1beta1.DestinationRuleList
			err := k8sClient.List(context.TODO(), &drList)
			if err != nil {
				return err
			}
			if len(drList.Items) > 0 {
				return fmt.Errorf("there are %d %s present but shouldn't", len(drList.Items), kind)
			}
		default:
			return fmt.Errorf("can't check if resource is present for undefined kind %s", kind)
		}
		return nil
	}, testcontext.GetRetryOpts()...)
}

func IstioResourceContainerHasRequiredVersion(ctx context.Context, containerName, kind, resourceName, namespace string) error {
	requiredVersion := strings.Join([]string{controllers.IstioVersion, controllers.IstioImageBase}, "-")

	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	return retry.Do(func() error {
		var object client.Object
		switch kind {
		case Deployment.String():
			object = &v1.Deployment{}
		case DaemonSet.String():
			object = &v1.DaemonSet{}
		default:
			return godog.ErrUndefined
		}
		err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: resourceName}, object)
		if err != nil {
			return err
		}

		switch kind {
		case Deployment.String():
			hasExpectedVersion := false
			for _, c := range object.(*v1.Deployment).Spec.Template.Spec.Containers {
				if c.Name != containerName {
					continue
				}
				deployedVersion, err := getVersionFromImageName(c.Image)
				if err != nil {
					return err
				}
				if deployedVersion != requiredVersion {
					return fmt.Errorf("container: %s kind: %s name: %s in namespace %s has version %s when required %s", containerName, kind, resourceName, namespace, deployedVersion, requiredVersion)
				}
				hasExpectedVersion = true
			}
			if !hasExpectedVersion {
				return fmt.Errorf("container: %s kind: %s name: %s in namespace %s not found", containerName, kind, resourceName, namespace)
			}
		case DaemonSet.String():
			hasExpectedVersion := false
			for _, c := range object.(*v1.DaemonSet).Spec.Template.Spec.Containers {
				if c.Name != containerName {
					continue
				}
				deployedVersion, err := getVersionFromImageName(c.Image)
				if err != nil {
					return err
				}
				if deployedVersion != requiredVersion {
					return fmt.Errorf("container: %s kind: %s name: %s in namespace %s has version %s when required %s", containerName, kind, resourceName, namespace, deployedVersion, requiredVersion)
				}
				hasExpectedVersion = true
			}
			if !hasExpectedVersion {
				return fmt.Errorf("container: %s kind: %s name: %s in namespace %s not found", containerName, kind, resourceName, namespace)
			}
		default:
			return godog.ErrUndefined
		}

		return nil
	}, testcontext.GetRetryOpts()...)
}
