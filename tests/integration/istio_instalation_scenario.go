package integration

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/tests/integration/manifests"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func initIstioScenarios(ctx *godog.ScenarioContext) {
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is ready`, resourceIsReady)
	ctx.Step(`^Istio CRD is installed$`, istioCRDIsInstalled)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" has status "([^"]*)"$`, istioCRInNamespaceHasStatus)
	ctx.Step(`^Istio CR "([^"]*)" is applied in namespace "([^"]*)"$`, istioCRIsAppliedInNamespace)
	ctx.Step(`^Namespace "([^"]*)" is "([^"]*)"$`, namespaceIsPresent)
	ctx.Step(`^Istio CRDs "([^"]*)" be present on cluster$`, istioCRDsBePresentOnCluster)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is deleted$`, istiosampleInNamespaceIsDeleted)
	ctx.Step(`^"([^"]*)" is not present on cluster$`, resourceNotPresent)
}

func resourceIsReady(kind, name, namespace string) error {
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
	}, retryOpts...)
}

func istioCRDIsInstalled() error {
	var crd unstructured.Unstructured
	crd.SetGroupVersionKind(CRDGroupVersionKind)
	return retry.Do(func() error {
		return k8sClient.Get(context.TODO(), types.NamespacedName{Name: "istios.operator.kyma-project.io"}, &crd)
	}, retryOpts...)
}

func istioCRInNamespaceHasStatus(name, namespace, status string) error {
	var cr istioCR.Istio
	return retry.Do(func() error {
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &cr)
		if err != nil {
			return err
		}
		if string(cr.Status.State) != status {
			return fmt.Errorf("status %s of Istio CR is not equal to %s", cr.Status.State, status)
		}
		return nil
	}, retryOpts...)
}

func istioCRIsAppliedInNamespace(name, namespace string) error {
	istioCr := istioCR.Istio{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return retry.Do(func() error {
		return k8sClient.Create(context.TODO(), &istioCr)
	}, retryOpts...)
}

func namespaceIsPresent(name, shouldBePresent string) error {
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
	}, retryOpts...)
}

func istioCRDsBePresentOnCluster(should string) error {
	shouldHave := true
	if should != "should" {
		shouldHave = false
	}
	lister, err := manifests.NewCRDListerFromFile(k8sClient, crdListPath)
	if err != nil {
		return err
	}
	return retry.Do(func() error {
		wrongs, err := lister.CheckForCRDs(context.TODO(), shouldHave)
		if err != nil {
			return err
		}
		if len(wrongs) > 0 {
			if shouldHave {
				return fmt.Errorf("CRDs %s where not present", strings.Join(wrongs, ";"))
			} else {
				return fmt.Errorf("CRDs %s where present", strings.Join(wrongs, ";"))
			}
		}
		return nil
	}, retryOpts...)
}

func istiosampleInNamespaceIsDeleted(kind, name, namespace string) error {
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
	default:
		return godog.ErrUndefined
	}
}

func resourceNotPresent(kind string) error {
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
		}
		return nil
	}, retryOpts...)
}
