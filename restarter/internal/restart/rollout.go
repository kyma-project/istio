package restart

import appsv1 "k8s.io/api/apps/v1"

func DeploymentRolledOut(d *appsv1.Deployment) bool {
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

func StatefulSetRolledOut(s *appsv1.StatefulSet) bool {
	if s.Generation > s.Status.ObservedGeneration {
		return false
	}
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

func DaemonSetRolledOut(ds *appsv1.DaemonSet) bool {
	if ds.Generation > ds.Status.ObservedGeneration {
		return false
	}
	return ds.Status.UpdatedNumberScheduled == ds.Status.DesiredNumberScheduled &&
		ds.Status.NumberAvailable == ds.Status.DesiredNumberScheduled
}

func ReplicaSetRolledOut(rs *appsv1.ReplicaSet) bool {
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
