package status

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

type executorPhaseManger struct {
	*v1alpha1.TiflowCluster
	cli       client.Client
	clientSet kubernetes.Interface
}

func NewExecutorPhaseManager(cli client.Client, clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) SyncPhaseManager {
	return &executorPhaseManger{
		tc,
		cli,
		clientSet,
	}
}

func setExecutorClusterStatusOnFirstReconcile(executorStatus *v1alpha1.ExecutorStatus) {
	InitExecutorClusterSyncTypesIfNeed(executorStatus)
	if executorStatus.Phase != "" {
		return
	}

	executorStatus.Phase = v1alpha1.ExecutorStarting
	executorStatus.Message = "Starting..., tiflow-executor on first reconcile. Just a moment"
	executorStatus.LastTransitionTime = metav1.Now()
	executorStatus.LastUpdateTime = metav1.Now()
	return
}

func InitExecutorClusterSyncTypesIfNeed(executorStatus *v1alpha1.ExecutorStatus) {
	if executorStatus.SyncTypes == nil {
		executorStatus.SyncTypes = []v1alpha1.ClusterSyncType{}
	}
	return
}

// SyncPhase
// todo: need to check for all OperatorActions or for just the 0th element
// This depends on our logic for updating Status
func (em *executorPhaseManger) SyncPhase() {
	executorStatus := em.GetExecutorStatus()

	if em.WithoutLocalExecutor() {
		executorStatus.Phase = v1alpha1.ExecutorRunning
		executorStatus.Message = "Ready..., without local tiflow-executor. Enjoying..."
		executorStatus.LastTransitionTime = metav1.Now()
		return
	}

	InitExecutorClusterSyncTypesIfNeed(executorStatus)

	if conditionIsTrue(v1alpha1.ExecutorSyncChecked, em.GetClusterConditions()) &&
		!em.syncExecutorPhaseFromCluster() {
		return
	}

	for _, sync := range executorStatus.SyncTypes {
		if sync.Status == v1alpha1.Completed {
			continue
		}

		switch sync.Status {
		case v1alpha1.Failed:
			executorStatus.Phase = v1alpha1.ExecutorFailed
		case v1alpha1.Unknown:
			executorStatus.Phase = v1alpha1.ExecutorUnknown
		default:
			executorStatus.Phase = sync.Name.GetExecutorClusterPhase()
		}

		executorStatus.Message = sync.Message
		executorStatus.LastTransitionTime = metav1.Now()
		return
	}

	executorStatus.Phase = v1alpha1.ExecutorRunning
	executorStatus.Message = "Ready..., tiflow-executor reconcile completed successfully. Enjoying..."
	executorStatus.LastTransitionTime = metav1.Now()
	return
}

func (em *executorPhaseManger) syncExecutorPhaseFromCluster() bool {
	if em.syncExecutorCreatePhase() {
		return true
	}

	if em.syncExecutorScalePhase() {
		return true
	}

	if em.syncExecutorUpgradePhase() {
		return true
	}

	return false

}
func (em *executorPhaseManger) syncExecutorCreatePhase() bool {
	syncTypes := em.GetExecutorSyncTypes()
	index := findPos(v1alpha1.CreateType, syncTypes)
	if index < 0 || syncTypes[index].Status == v1alpha1.Completed {
		return false
	}

	if em.ExecutorStsDesiredReplicas() == em.ExecutorStsReadyReplicas() {
		Completed(v1alpha1.CreateType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, "executor creating completed")
		return true
	}

	return false
}

func (em *executorPhaseManger) syncExecutorScalePhase() bool {
	syncTypes := em.GetExecutorSyncTypes()

	index := findPos(v1alpha1.ScaleOutType, syncTypes)
	if index >= 0 && syncTypes[index].Status != v1alpha1.Completed {
		if em.ExecutorStsDesiredReplicas() == em.ExecutorStsCurrentReplicas() {
			Completed(v1alpha1.ScaleOutType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, "executor scaling out completed")
			return true
		}
		return false
	}

	index = findPos(v1alpha1.ScaleInType, syncTypes)
	if index >= 0 && syncTypes[index].Status != v1alpha1.Completed {
		if em.ExecutorStsDesiredReplicas() == em.ExecutorStsCurrentReplicas() {
			Completed(v1alpha1.ScaleInType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, "executor scaling in completed")
			return true
		}
		return false
	}

	return false
}

// syncExecutorUpgradePhase return true indicates a change in the current phase
func (em *executorPhaseManger) syncExecutorUpgradePhase() bool {
	syncTypes := em.GetExecutorSyncTypes()

	index := findPos(v1alpha1.UpgradeType, syncTypes)

	if index < 0 || syncTypes[index].Status == v1alpha1.Completed {
		return false
	}

	ns := em.GetNamespace()
	tcName := em.GetName()

	sts, err := em.clientSet.AppsV1().StatefulSets(ns).
		Get(context.TODO(), executorMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("executor [%s/%s] upgrade phase: can not get statefulSet", ns, tcName)
		Unknown(v1alpha1.UpgradeType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, message)
		return true
	}

	if isUpgrading(sts) {
		return false
	}

	instanceName := em.GetInstanceName()
	selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	if err != nil {
		message := fmt.Sprintf("executor [%s/%s] upgrade phase: converting statefulSet selector error: %v",
			ns, instanceName, err)
		Failed(v1alpha1.UpgradeType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, message)
		return true
	}

	executorPods, err := em.clientSet.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		message := fmt.Sprintf("executor [%s/%s] upgrade phase: listing executor's pods error: %v",
			ns, instanceName, err)
		Failed(v1alpha1.UpgradeType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, message)
		return true
	}

	// todo: more gracefully
	for _, pod := range executorPods.Items {
		revisionHash, exist := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
		if !exist {
			message := fmt.Sprintf("executor [%s/%s] upgrade phase: has no label ControllerRevisionHashLabelKey",
				ns, tcName)
			Failed(v1alpha1.UpgradeType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, message)
			return true
		}
		if revisionHash != em.GetExecutorStatus().StatefulSet.UpdateRevision {
			return false
		}
	}

	Completed(v1alpha1.UpgradeType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, "executor upgrading  completed")
	return true
}
