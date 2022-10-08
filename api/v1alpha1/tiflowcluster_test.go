package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestTiflowMasterIsAvailable(t *testing.T) {
	type testcase struct {
		name            string
		update          func(cluster *TiflowCluster)
		expectAvailable bool
	}
	testFn := func(test *testcase) {
		t.Log(test.name)

		tc := newTiflowCluster()
		test.update(tc)
		require.Equal(t, test.expectAvailable, tc.MasterIsAvailable())
	}
	tests := []testcase{
		{
			name: "tiflow-master leader is not up",
			update: func(tc *TiflowCluster) {
				tc.Status.Master.Members = map[string]MasterMember{}
				tc.Status.Master.Leader = MasterMember{}
			},
			expectAvailable: false,
		},
		{
			name: "tiflow-master is available",
			update: func(tc *TiflowCluster) {
				tc.Status.Master.Members = map[string]MasterMember{
					"basic-tiflow-master-0": {Name: "basic-tiflow-master-0", Id: "basic-tiflow-master-0.default-8c6be692"},
					"basic-tiflow-master-1": {Name: "basic-tiflow-master-1", Id: "basic-tiflow-master-1.default-5fd0afd7", IsLeader: true},
					"basic-tiflow-master-2": {Name: "basic-tiflow-master-2", Id: "basic-tiflow-master-2.default-6dc32d1f"},
				}
				tc.Status.Master.Leader = tc.Status.Master.Members["basic-tiflow-master-1"]
			},
			expectAvailable: true,
		},
	}

	for i := range tests {
		testFn(&tests[i])
	}
}

func newTiflowCluster() *TiflowCluster {
	return &TiflowCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TiflowCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basic",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: TiflowClusterSpec{
			Master: &MasterSpec{
				Replicas: 3,
			},
			Executor: &ExecutorSpec{
				Replicas:    3,
				StorageSize: "10G",
			},
		},
	}
}
