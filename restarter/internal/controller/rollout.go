package controller

import appsv1 "k8s.io/api/apps/v1"

// The rolledOut predicates mirror `kubectl rollout status`: a workload is
// "rolled out" when its latest spec is fully observed and every replica runs the
// updated revision. The restarter must not restart a workload whose previous
// restart is still rolling out — old-revision pods still present mid-rollout
// would be read as drift by pod-state evaluators and re-trigger the restart
// before it settles. A rollout is in progress whenever Generation has outrun
// Status.ObservedGeneration, so the generation check uses > (not >=).

func deploymentRolledOut(d *appsv1.Deployment) bool {
	if d.Generation > d.Status.ObservedGeneration {
		return false
	}
	if d.Spec.Replicas == nil {
		return false
	}
	want := *d.Spec.Replicas
	return d.Status.UpdatedReplicas == want &&
		d.Status.Replicas == want &&
		d.Status.AvailableReplicas == want
}

func statefulSetRolledOut(s *appsv1.StatefulSet) bool {
	if s.Generation > s.Status.ObservedGeneration {
		return false
	}
	// UpdateRevision == CurrentRevision is the authoritative "no old-revision
	// pods remain" signal; replica counts alone miss a partitioned update.
	if s.Status.UpdateRevision != s.Status.CurrentRevision {
		return false
	}
	want := int32(1)
	if s.Spec.Replicas != nil {
		want = *s.Spec.Replicas
	}
	return s.Status.UpdatedReplicas == want &&
		s.Status.CurrentReplicas == want &&
		s.Status.ReadyReplicas == want &&
		s.Status.AvailableReplicas == want
}

func daemonSetRolledOut(ds *appsv1.DaemonSet) bool {
	if ds.Generation > ds.Status.ObservedGeneration {
		return false
	}
	return ds.Status.UpdatedNumberScheduled == ds.Status.DesiredNumberScheduled &&
		ds.Status.NumberAvailable == ds.Status.DesiredNumberScheduled
}

func replicaSetRolledOut(rs *appsv1.ReplicaSet) bool {
	if rs.Generation > rs.Status.ObservedGeneration {
		return false
	}
	if rs.Spec.Replicas == nil {
		return false
	}
	want := *rs.Spec.Replicas
	return rs.Status.Replicas == want &&
		rs.Status.AvailableReplicas == want
}
