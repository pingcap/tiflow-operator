package status

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

type ExecutorPhaseManger struct {
	*v1alpha1.TiflowCluster
	cli       client.Client
	clientSet kubernetes.Interface
}

func NewExecutorPhaseManager(cli client.Client, clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) SyncPhaseManager {
	return &ExecutorPhaseManger{
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
func (em *ExecutorPhaseManger) SyncPhase() {
	executorStatus := em.GetExecutorStatus()
	InitExecutorClusterSyncTypesIfNeed(executorStatus)

	if !em.syncExecutorPhaseFromCluster() {
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

func (em *ExecutorPhaseManger) syncExecutorPhaseFromCluster() bool {
	if em.syncExecutorCreatPhase() {
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
func (em *ExecutorPhaseManger) syncExecutorCreatPhase() bool {
	syncTypes := em.GetExecutorSyncTypes()
	index := findPos(v1alpha1.CreateType, syncTypes)
	if index < 0 || syncTypes[index].Status == v1alpha1.Completed {
		return false
	}

	if em.ExecutorStsDesiredReplicas() == em.ExecutorStsCurrentReplicas() {
		Complied(v1alpha1.CreateType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, "executor creating completed")
		return true
	}

	return false
}

func (em *ExecutorPhaseManger) syncExecutorScalePhase() bool {
	syncTypes := em.GetExecutorSyncTypes()

	index := findPos(v1alpha1.ScaleOutType, syncTypes)
	if index >= 0 && syncTypes[index].Status != v1alpha1.Completed {
		if em.ExecutorStsDesiredReplicas() == em.ExecutorStsCurrentReplicas() {
			Complied(v1alpha1.ScaleOutType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, "executor scaling out completed")
			return true
		}
		return false
	}

	index = findPos(v1alpha1.ScaleInType, syncTypes)
	if index >= 0 && syncTypes[index].Status != v1alpha1.Completed {
		if em.ExecutorStsDesiredReplicas() == em.ExecutorStsCurrentReplicas() {
			Complied(v1alpha1.ScaleInType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, "executor scaling in completed")
			return true
		}
		return false
	}

	return false
}

func (em *ExecutorPhaseManger) syncExecutorUpgradePhase() bool {
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
		Unknown(v1alpha1.UpgradeType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, "executor upgrading  unknown")
		return false
	}

	if isUpgrading(sts) {
		return false
	}

	instanceName := em.GetInstanceName()
	selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	if err != nil {
		klog.Infof("executor [%s/%s] status converting statefulSet selector error: %v",
			ns, instanceName, err)
		Failed(v1alpha1.UpgradeType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, "executor upgrading  failed")
		return false
	}

	executorPods, err := em.clientSet.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		klog.Infof("executor [%s/%s] status listing executor's pods error: %v",
			ns, instanceName, err)
		Failed(v1alpha1.UpgradeType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, "executor upgrading  failed")
		return false
	}

	// todo: more gracefully
	for _, pod := range executorPods.Items {
		revisionHash, exist := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
		if !exist && revisionHash == em.GetExecutorStatus().StatefulSet.UpdateRevision {
			Complied(v1alpha1.UpgradeType, em.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType, "executor upgrading  completed")
			return true
		}
	}

	return false
}
