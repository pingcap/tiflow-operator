package testutil

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/manager/member"
	testenv "github.com/pingcap/tiflow-operator/pkg/testutil/env"
	"github.com/pingcap/tiflow-operator/pkg/tiflowapi"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RequireClusterToBeReadyEventuallyTimeout tests to see if a statefulset has started correctly and
// all the pods are ready.
func RequireClusterToBeReadyEventuallyTimeout(t *testing.T, sb testenv.Sandbox, b ClusterBuilder, createTime metav1.Time, timeout time.Duration) {
	t.Helper()
	cluster := b.Cr()

	require.NoError(t, wait.Poll(10*time.Second, timeout, func() (bool, error) {
		tfc := &v1alpha1.TiflowCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: cluster.Name,
			},
		}
		found, err := fetchClientObject(sb, tfc)
		if err != nil {
			t.Logf("error fetching tiflow cluster %s", cluster.Name)
			return false, err
		}

		if !found {
			t.Logf("tiflow cluster %s is not found", cluster.Name)
			return false, nil
		}

		if !tiflowClusterIsReady(t, tfc, createTime) {
			var statusBytes []byte
			statusBytes, err = json.Marshal(cluster.Status)
			if err != nil {
				return false, err
			}
			t.Logf("tiflow cluster %s is not ready, status: %s", cluster.Name, statusBytes)
			if err = logPods(context.TODO(), tfc, sb, t); err != nil {
				return false, err
			}
			return false, nil
		}

		return true, nil
	}))

	t.Log("tiflow cluster is ready")
}

func fetchClientObject(sb testenv.Sandbox, obj client.Object) (bool, error) {
	err := sb.Get(obj)
	if err != nil && apierrors.IsNotFound(err) {
		return false, nil
	}
	return true, err
}

func tiflowClusterIsReady(t *testing.T, tc *v1alpha1.TiflowCluster, createTime metav1.Time) bool {
	t.Helper()
	if tc.Status.Master.StatefulSet.ReadyReplicas != tc.Spec.Master.Replicas ||
		tc.Status.Executor.StatefulSet.ReadyReplicas != tc.Spec.Executor.Replicas {
		t.Logf("tiflow cluster %s is not ready because replicas are not ready, master replicas: [%d/%d], "+
			"executor replicas: [%d/%d]", tc.Name, tc.MasterStsReadyReplicas(), tc.Spec.Master.Replicas,
			tc.ExecutorStsReadyReplicas(), tc.Spec.Executor.Replicas)
		return false
	}
	if createTime.Time.After(tc.Status.LastTransitionTime.Time) || tc.Status.ClusterPhase != v1alpha1.ClusterRunning ||
		createTime.Time.After(tc.Status.Master.LastTransitionTime.Time) || tc.Status.Master.Phase != v1alpha1.MasterRunning ||
		createTime.Time.After(tc.Status.Executor.LastTransitionTime.Time) || tc.Status.Executor.Phase != v1alpha1.ExecutorRunning {
		t.Logf("tiflow cluster %s is not ready because cluster stage is not running", tc.Name)
		return false
	}
	scheme := "http"
	if tc.IsClusterTLSEnabled() {
		scheme = "https"
	}
	tcName := tc.Name
	tcNamespace := tc.Namespace
	if tc.Spec.Cluster != nil && tc.Spec.Master == nil {
		tcName = tc.Spec.Cluster.Name
		tcNamespace = tc.Spec.Cluster.Namespace
	}
	if !strings.EqualFold(tc.Status.Master.ServerAddress, tiflowapi.MasterClientURL(tcNamespace, tcName, "", scheme)) {
		t.Logf("tiflow cluster %s is not ready because cluster server address %s is not correct", tc.Name, tc.Status.Master.ServerAddress)
		return false
	}
	masterSet := make(map[string]struct{})
	executorSet := make(map[string]struct{})
	for i := int32(0); i < tc.Spec.Master.Replicas; i++ {
		masterSet[member.TiflowMasterPodName(tc.Name, i)+"."+tc.Namespace] = struct{}{}
	}
	for i := int32(0); i < tc.Spec.Executor.Replicas; i++ {
		executorSet[member.TiflowExecutorPodName(tc.Name, i)+"."+tc.Namespace] = struct{}{}
	}
	for masterName := range tc.Status.Master.Members {
		if _, ok := masterSet[masterName]; !ok {
			t.Logf("tiflow cluster %s is not ready because master %s doesn't exist", tc.Name, masterName)
			return false
		}
	}
	for executorName := range tc.Status.Executor.Members {
		if _, ok := executorSet[executorName]; !ok {
			t.Logf("tiflow cluster %s is not ready because executor %s doesn't exist", tc.Name, executorName)
			return false
		}
	}

	return true
}

// RequireDeploymentToBeReadyEventuallyTimeout tests to see if a statefulset has started correctly and
// all the pods are ready.
func RequireDeploymentToBeReadyEventuallyTimeout(t *testing.T, sb testenv.Sandbox, name string, timeout time.Duration) {
	t.Helper()
	require.NoError(t, wait.Poll(10*time.Second, timeout, func() (bool, error) {
		dp := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}

		found, err := fetchClientObject(sb, dp)
		if err != nil {
			t.Logf("error fetching deployment")
			return false, err
		}

		if !found {
			t.Logf("deployment is not found")
			return false, nil
		}

		if !deploymentIsReady(dp) {
			t.Logf("deployment is not ready")
			return false, nil
		}

		return true, nil
	}))

	t.Log("Deployment is ready")
}

func deploymentIsReady(dp *appsv1.Deployment) bool {
	return dp.Status.ReadyReplicas == *dp.Spec.Replicas
}

func logPods(ctx context.Context, tfc *v1alpha1.TiflowCluster,
	sb testenv.Sandbox, t *testing.T) error {
	t.Helper()
	// create a new clientset to talk to k8s
	clientset, err := kubernetes.NewForConfig(sb.Mgr.GetConfig())
	if err != nil {
		return err
	}

	options := metav1.ListOptions{}

	// Get all pods
	podList, err := clientset.CoreV1().Pods(tfc.Namespace).List(ctx, options)
	if err != nil {
		return err
	}

	if len(podList.Items) == 0 {
		t.Log("no pods found")
	}

	// Print out pretty into on the Pods
	for _, podInfo := range (*podList).Items {
		t.Logf("pods-name=%v\n", podInfo.Name)
		t.Logf("pods-status=%v\n", podInfo.Status.Phase)
		t.Logf("pods-condition=%v\n", podInfo.Status.Conditions)
	}

	return nil
}
