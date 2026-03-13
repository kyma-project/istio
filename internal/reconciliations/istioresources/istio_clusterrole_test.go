package istioresources

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

func getClusterRole(ctx context.Context, c client.Client, name string) (*rbacv1.ClusterRole, error) {
	var cr rbacv1.ClusterRole
	if err := c.Get(ctx, ctrlclient.ObjectKey{Name: name}, &cr); err != nil {
		return nil, err
	}
	return &cr, nil
}

var _ = Describe("ClusterRoles", func() {
	owner := metav1.OwnerReference{
		APIVersion: "operator.kyma-project.io/v1alpha2",
		Kind:       "Istio",
		Name:       "owner-name",
		UID:        "owner-uid",
	}

	Context("creation", func() {
		It("should create both ClusterRoles when they do not exist", func() {
			// given
			c := createFakeClient()
			reconciler := NewClusterRolesReconciler(false)

			// when
			op, err := reconciler.reconcile(context.Background(), c, owner, nil)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(op).To(Equal(controllerutil.OperationResultCreated))

			editCR, err := getClusterRole(context.Background(), c, "kyma-istio-resources-edit")
			Expect(err).To(Not(HaveOccurred()))
			Expect(editCR).ToNot(BeNil())
			Expect(editCR.Labels).To(HaveKeyWithValue("rbac.authorization.k8s.io/aggregate-to-edit", "true"))
			Expect(editCR.OwnerReferences).To(ContainElement(owner))

			viewCR, err := getClusterRole(context.Background(), c, "kyma-istio-resources-view")
			Expect(err).To(Not(HaveOccurred()))
			Expect(viewCR).ToNot(BeNil())
			Expect(viewCR.Labels).To(HaveKeyWithValue("rbac.authorization.k8s.io/aggregate-to-view", "true"))
			Expect(viewCR.OwnerReferences).To(ContainElement(owner))
		})
	})

	Context("update", func() {
		It("should update an existing ClusterRole when its labels differ", func() {
			// given — pre-create both ClusterRoles; remove labels on edit to simulate drift
			var editCR rbacv1.ClusterRole
			Expect(yaml.Unmarshal(editClusterRole, &editCR)).Should(Succeed())
			editCR.Labels = map[string]string{} // wipe labels to simulate drift

			var viewCR rbacv1.ClusterRole
			Expect(yaml.Unmarshal(viewClusterRole, &viewCR)).Should(Succeed())

			c := createFakeClient(&editCR, &viewCR)
			reconciler := NewClusterRolesReconciler(false)

			// when
			op, err := reconciler.reconcile(context.Background(), c, owner, nil)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(op).To(Equal(controllerutil.OperationResultUpdated))

			updated, err := getClusterRole(context.Background(), c, "kyma-istio-resources-edit")
			Expect(err).To(Not(HaveOccurred()))
			Expect(updated.Labels).To(HaveKeyWithValue("rbac.authorization.k8s.io/aggregate-to-edit", "true"))
			Expect(updated.OwnerReferences).To(ContainElement(owner))
		})

		It("should return unchanged when reconciling twice with same data", func() {
			// given
			c := createFakeClient()
			reconciler := NewClusterRolesReconciler(false)

			// first reconcile — creates
			op, err := reconciler.reconcile(context.Background(), c, owner, nil)
			Expect(err).To(Not(HaveOccurred()))
			Expect(op).To(Equal(controllerutil.OperationResultCreated))

			// when — second reconcile
			op, err = reconciler.reconcile(context.Background(), c, owner, nil)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(op).To(Equal(controllerutil.OperationResultNone))
		})

		It("should restore owner reference if it was removed", func() {
			// given — pre-create both ClusterRoles; remove owner ref on view to simulate drift
			var editCR rbacv1.ClusterRole
			Expect(yaml.Unmarshal(editClusterRole, &editCR)).Should(Succeed())

			var viewCR rbacv1.ClusterRole
			Expect(yaml.Unmarshal(viewClusterRole, &viewCR)).Should(Succeed())
			viewCR.OwnerReferences = nil

			c := createFakeClient(&editCR, &viewCR)
			reconciler := NewClusterRolesReconciler(false)

			// when
			op, err := reconciler.reconcile(context.Background(), c, owner, nil)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(op).To(Equal(controllerutil.OperationResultUpdated))

			updated, err := getClusterRole(context.Background(), c, "kyma-istio-resources-view")
			Expect(err).To(Not(HaveOccurred()))
			Expect(updated.OwnerReferences).To(ContainElement(owner))
		})
	})

	Context("deletion", func() {
		It("should delete both ClusterRoles when they exist", func() {
			// given — create roles first
			c := createFakeClient()
			creator := NewClusterRolesReconciler(false)
			_, err := creator.reconcile(context.Background(), c, owner, nil)
			Expect(err).To(Not(HaveOccurred()))

			// sanity check: roles exist
			_, err = getClusterRole(context.Background(), c, "kyma-istio-resources-edit")
			Expect(err).To(Not(HaveOccurred()))
			_, err = getClusterRole(context.Background(), c, "kyma-istio-resources-view")
			Expect(err).To(Not(HaveOccurred()))

			// when
			deleter := NewClusterRolesReconciler(true)
			op, err := deleter.reconcile(context.Background(), c, owner, nil)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(op).To(Equal(controllerutil.OperationResultUpdated))

			_, editErr := getClusterRole(context.Background(), c, "kyma-istio-resources-edit")
			Expect(editErr).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(editErr)).To(BeTrue())

			_, viewErr := getClusterRole(context.Background(), c, "kyma-istio-resources-view")
			Expect(viewErr).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(viewErr)).To(BeTrue())
		})
		It("should succeed when ClusterRoles do not exist (no-op)", func() {
			// given
			c := createFakeClient()
			deleter := NewClusterRolesReconciler(true)

			// when
			op, err := deleter.reconcile(context.Background(), c, owner, nil)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(op).To(Equal(controllerutil.OperationResultNone))
		})
	})
})
