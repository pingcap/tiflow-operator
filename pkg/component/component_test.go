package component

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

func TestTiflowComponentAccessor(t *testing.T) {
	type testcase struct {
		name      string
		cluster   *v1alpha1.TiflowClusterSpec
		component *v1alpha1.ComponentSpec
		expectFn  func(ComponentAccessor)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := &v1alpha1.TiflowCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: *test.cluster,
		}

		accessor := buildTiflowClusterComponentAccessor(ComponentTiflowMaster, tc, test.component)
		test.expectFn(accessor)
	}
	affinity := &corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
				TopologyKey: "rack",
			}},
		},
	}
	toleration1 := corev1.Toleration{
		Key: "k1",
	}
	toleration2 := corev1.Toleration{
		Key: "k2",
	}
	tests := []testcase{
		{
			name: "use cluster-level defaults",
			cluster: &v1alpha1.TiflowClusterSpec{
				ImagePullPolicy:   corev1.PullNever,
				ImagePullSecrets:  []corev1.LocalObjectReference{{Name: "image-pull-secret"}},
				HostNetwork:       pointer.Bool(true),
				Affinity:          affinity,
				PriorityClassName: pointer.String("test"),
			},
			component: &v1alpha1.ComponentSpec{},
			expectFn: func(a ComponentAccessor) {
				require.Equal(t, corev1.PullNever, a.ImagePullPolicy())
				require.Equal(t, []corev1.LocalObjectReference{{Name: "image-pull-secret"}}, a.ImagePullSecrets())
				require.True(t, a.HostNetwork())
				require.Equal(t, affinity, a.Affinity())
				require.Equal(t, "test", *a.PriorityClassName())
			},
		},
		{
			name: "override at component-level",
			cluster: &v1alpha1.TiflowClusterSpec{
				ImagePullPolicy:   corev1.PullNever,
				ImagePullSecrets:  []corev1.LocalObjectReference{{Name: "cluster-level-secret"}},
				HostNetwork:       pointer.Bool(true),
				Affinity:          nil,
				PriorityClassName: pointer.String("test"),
			},
			component: &v1alpha1.ComponentSpec{
				ImagePullSecrets:  []corev1.LocalObjectReference{{Name: "component-level-secret"}},
				ImagePullPolicy:   func() *corev1.PullPolicy { a := corev1.PullAlways; return &a }(),
				HostNetwork:       pointer.Bool(false),
				Affinity:          affinity,
				PriorityClassName: pointer.String("override"),
			},
			expectFn: func(a ComponentAccessor) {
				require.Equal(t, corev1.PullAlways, a.ImagePullPolicy())
				require.Equal(t, []corev1.LocalObjectReference{{Name: "component-level-secret"}}, a.ImagePullSecrets())
				require.False(t, a.HostNetwork())
				require.Equal(t, affinity, a.Affinity())
				require.Equal(t, "override", *a.PriorityClassName())
			},
		},
		{
			name: "node selector merge",
			cluster: &v1alpha1.TiflowClusterSpec{
				NodeSelector: map[string]string{
					"k1": "v1",
				},
			},
			component: &v1alpha1.ComponentSpec{
				NodeSelector: map[string]string{
					"k1": "v2",
					"k3": "v3",
				},
			},
			expectFn: func(a ComponentAccessor) {
				require.Equal(t, map[string]string{
					"k1": "v2",
					"k3": "v3",
				}, a.NodeSelector())
			},
		},
		{
			name: "labels merge",
			cluster: &v1alpha1.TiflowClusterSpec{
				Labels: map[string]string{
					"k1": "v1",
				},
			},
			component: &v1alpha1.ComponentSpec{
				Labels: map[string]string{
					"k1": "v2",
					"k3": "v3",
				},
			},
			expectFn: func(a ComponentAccessor) {
				require.Equal(t, map[string]string{
					"k1": "v2",
					"k3": "v3",
				}, a.Labels())
			},
		},
		{
			name: "annotations merge",
			cluster: &v1alpha1.TiflowClusterSpec{
				Annotations: map[string]string{
					"k1": "v1",
				},
			},
			component: &v1alpha1.ComponentSpec{
				Annotations: map[string]string{
					"k1": "v2",
					"k3": "v3",
				},
			},
			expectFn: func(a ComponentAccessor) {
				require.Equal(t, map[string]string{
					"k1": "v2",
					"k3": "v3",
				}, a.Annotations())
			},
		},
		{
			name: "tolerations merge",
			cluster: &v1alpha1.TiflowClusterSpec{
				Tolerations: []corev1.Toleration{toleration1},
			},
			component: &v1alpha1.ComponentSpec{
				Tolerations: []corev1.Toleration{toleration2},
			},
			expectFn: func(a ComponentAccessor) {
				require.Contains(t, a.Tolerations(), toleration2)
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}
