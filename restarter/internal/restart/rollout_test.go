package restart

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeploymentRolledOut(t *testing.T) {
	tests := []struct {
		name string
		dep  *appsv1.Deployment
		want bool
	}{
		{
			name: "generation ahead of observed",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 3},
				Spec:       appsv1.DeploymentSpec{Replicas: new(int32(2))},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 2,
					UpdatedReplicas:    2,
					Replicas:           2,
					AvailableReplicas:  2,
				},
			},
			want: false,
		},
		{
			name: "nil replicas",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec:       appsv1.DeploymentSpec{Replicas: nil},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 1,
					UpdatedReplicas:    1,
					Replicas:           1,
					AvailableReplicas:  1,
				},
			},
			want: false,
		},
		{
			name: "fully rolled out",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 4},
				Spec:       appsv1.DeploymentSpec{Replicas: new(int32(3))},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 4,
					UpdatedReplicas:    3,
					Replicas:           3,
					AvailableReplicas:  3,
				},
			},
			want: true,
		},
		{
			name: "updated replicas lagging",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 4},
				Spec:       appsv1.DeploymentSpec{Replicas: new(int32(3))},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 4,
					UpdatedReplicas:    2,
					Replicas:           3,
					AvailableReplicas:  3,
				},
			},
			want: false,
		},
		{
			name: "available replicas lagging",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 4},
				Spec:       appsv1.DeploymentSpec{Replicas: new(int32(3))},
				Status: appsv1.DeploymentStatus{
					ObservedGeneration: 4,
					UpdatedReplicas:    3,
					Replicas:           3,
					AvailableReplicas:  1,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeploymentRolledOut(tt.dep); got != tt.want {
				t.Errorf("DeploymentRolledOut() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatefulSetRolledOut(t *testing.T) {
	tests := []struct {
		name string
		sts  *appsv1.StatefulSet
		want bool
	}{
		{
			name: "generation ahead of observed",
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 3},
				Spec:       appsv1.StatefulSetSpec{Replicas: new(int32(2))},
				Status: appsv1.StatefulSetStatus{
					ObservedGeneration: 2,
					UpdateRevision:     "rev-2",
					CurrentRevision:    "rev-2",
					UpdatedReplicas:    2,
					CurrentReplicas:    2,
					ReadyReplicas:      2,
					AvailableReplicas:  2,
				},
			},
			want: false,
		},
		{
			name: "update revision differs from current",
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Spec:       appsv1.StatefulSetSpec{Replicas: new(int32(2))},
				Status: appsv1.StatefulSetStatus{
					ObservedGeneration: 2,
					UpdateRevision:     "rev-3",
					CurrentRevision:    "rev-2",
					UpdatedReplicas:    2,
					CurrentReplicas:    2,
					ReadyReplicas:      2,
					AvailableReplicas:  2,
				},
			},
			want: false,
		},
		{
			name: "fully rolled out",
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 5},
				Spec:       appsv1.StatefulSetSpec{Replicas: new(int32(3))},
				Status: appsv1.StatefulSetStatus{
					ObservedGeneration: 5,
					UpdateRevision:     "rev-5",
					CurrentRevision:    "rev-5",
					UpdatedReplicas:    3,
					CurrentReplicas:    3,
					ReadyReplicas:      3,
					AvailableReplicas:  3,
				},
			},
			want: true,
		},
		{
			name: "nil replicas defaults want to 1 and matches",
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec:       appsv1.StatefulSetSpec{Replicas: nil},
				Status: appsv1.StatefulSetStatus{
					ObservedGeneration: 1,
					UpdateRevision:     "rev-1",
					CurrentRevision:    "rev-1",
					UpdatedReplicas:    1,
					CurrentReplicas:    1,
					ReadyReplicas:      1,
					AvailableReplicas:  1,
				},
			},
			want: true,
		},
		{
			name: "nil replicas defaults want to 1 but status is zero",
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec:       appsv1.StatefulSetSpec{Replicas: nil},
				Status: appsv1.StatefulSetStatus{
					ObservedGeneration: 1,
					UpdateRevision:     "rev-1",
					CurrentRevision:    "rev-1",
					UpdatedReplicas:    0,
					CurrentReplicas:    0,
					ReadyReplicas:      0,
					AvailableReplicas:  0,
				},
			},
			want: false,
		},
		{
			name: "ready replicas lagging",
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 5},
				Spec:       appsv1.StatefulSetSpec{Replicas: new(int32(3))},
				Status: appsv1.StatefulSetStatus{
					ObservedGeneration: 5,
					UpdateRevision:     "rev-5",
					CurrentRevision:    "rev-5",
					UpdatedReplicas:    3,
					CurrentReplicas:    3,
					ReadyReplicas:      2,
					AvailableReplicas:  3,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StatefulSetRolledOut(tt.sts); got != tt.want {
				t.Errorf("StatefulSetRolledOut() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDaemonSetRolledOut(t *testing.T) {
	tests := []struct {
		name string
		ds   *appsv1.DaemonSet
		want bool
	}{
		{
			name: "generation ahead of observed",
			ds: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 3},
				Status: appsv1.DaemonSetStatus{
					ObservedGeneration:     2,
					DesiredNumberScheduled: 4,
					UpdatedNumberScheduled: 4,
					NumberAvailable:        4,
				},
			},
			want: false,
		},
		{
			name: "fully rolled out",
			ds: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Status: appsv1.DaemonSetStatus{
					ObservedGeneration:     2,
					DesiredNumberScheduled: 4,
					UpdatedNumberScheduled: 4,
					NumberAvailable:        4,
				},
			},
			want: true,
		},
		{
			name: "updated scheduled lagging",
			ds: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Status: appsv1.DaemonSetStatus{
					ObservedGeneration:     2,
					DesiredNumberScheduled: 4,
					UpdatedNumberScheduled: 3,
					NumberAvailable:        4,
				},
			},
			want: false,
		},
		{
			name: "number available lagging",
			ds: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Status: appsv1.DaemonSetStatus{
					ObservedGeneration:     2,
					DesiredNumberScheduled: 4,
					UpdatedNumberScheduled: 4,
					NumberAvailable:        2,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DaemonSetRolledOut(tt.ds); got != tt.want {
				t.Errorf("DaemonSetRolledOut() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplicaSetRolledOut(t *testing.T) {
	tests := []struct {
		name string
		rs   *appsv1.ReplicaSet
		want bool
	}{
		{
			name: "generation ahead of observed",
			rs: &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 3},
				Spec:       appsv1.ReplicaSetSpec{Replicas: new(int32(2))},
				Status: appsv1.ReplicaSetStatus{
					ObservedGeneration: 2,
					Replicas:           2,
					AvailableReplicas:  2,
				},
			},
			want: false,
		},
		{
			name: "nil replicas",
			rs: &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec:       appsv1.ReplicaSetSpec{Replicas: nil},
				Status: appsv1.ReplicaSetStatus{
					ObservedGeneration: 1,
					Replicas:           1,
					AvailableReplicas:  1,
				},
			},
			want: false,
		},
		{
			name: "fully rolled out",
			rs: &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Spec:       appsv1.ReplicaSetSpec{Replicas: new(int32(3))},
				Status: appsv1.ReplicaSetStatus{
					ObservedGeneration: 2,
					Replicas:           3,
					AvailableReplicas:  3,
				},
			},
			want: true,
		},
		{
			name: "available replicas lagging",
			rs: &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Spec:       appsv1.ReplicaSetSpec{Replicas: new(int32(3))},
				Status: appsv1.ReplicaSetStatus{
					ObservedGeneration: 2,
					Replicas:           3,
					AvailableReplicas:  1,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReplicaSetRolledOut(tt.rs); got != tt.want {
				t.Errorf("ReplicaSetRolledOut() = %v, want %v", got, tt.want)
			}
		})
	}
}
