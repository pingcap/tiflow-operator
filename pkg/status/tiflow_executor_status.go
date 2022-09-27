package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

type ExecutorSyncTypeManger struct {
	Status *v1alpha1.ExecutorStatus
}

func NewExecutorSyncTypeManager(status *v1alpha1.ExecutorStatus) SyncTypeManager {
	return &ExecutorSyncTypeManger{
		Status: status,
	}
}

func (em *ExecutorSyncTypeManger) Ongoing(syncName v1alpha1.SyncTypeName, message string) {
	em.setExecutorSyncTypeStatus(syncName, v1alpha1.Ongoing, message, metav1.Now())
}

func (em *ExecutorSyncTypeManger) Complied(syncName v1alpha1.SyncTypeName, message string) {
	em.setExecutorSyncTypeStatus(syncName, v1alpha1.Completed, message, metav1.Now())
}

func (em *ExecutorSyncTypeManger) Unknown(syncName v1alpha1.SyncTypeName, message string) {
	em.setExecutorSyncTypeStatus(syncName, v1alpha1.Unknown, message, metav1.Now())
}

func (em *ExecutorSyncTypeManger) Failed(syncName v1alpha1.SyncTypeName, message string) {
	em.setExecutorSyncTypeStatus(syncName, v1alpha1.Failed, message, metav1.Now())
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

// UpdateClusterStatus
// todo: need to check for all OperatorActions or for just the 0th element
// This depends on our logic for updating Status
func (em *ExecutorSyncTypeManger) UpdateClusterStatus(clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) {
	executorStatus := &tc.Status.Executor
	InitExecutorClusterSyncTypesIfNeed(executorStatus)

	em.syncExecutorPhaseFromCluster(clientSet, tc)

	for _, sync := range executorStatus.SyncTypes {
		if sync.Status == v1alpha1.Completed {
			continue
		}

		switch sync.Status {
		case v1alpha1.Failed:
			executorStatus.Phase = v1alpha1.ExecutorFailed
		case v1alpha1.Unknown:
			executorStatus.Phase = v1alpha1.ExecutorUnknown
		case v1alpha1.Completed:
			executorStatus.Phase = v1alpha1.ExecutorRunning
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

func (em *ExecutorSyncTypeManger) setExecutorSyncTypeStatus(syncName v1alpha1.SyncTypeName, syncStatus v1alpha1.SyncTypeStatus, message string, now metav1.Time) {
	sync := em.findOrCreateExecutorSyncType(syncName, message)
	sync.Status = syncStatus
	sync.LastUpdateTime = now
}

func (em *ExecutorSyncTypeManger) findOrCreateExecutorSyncType(syncName v1alpha1.SyncTypeName, message string) *v1alpha1.ClusterSyncType {
	pos := findPos(syncName, em.Status.SyncTypes)
	if pos >= 0 {
		em.Status.SyncTypes[pos].Message = message
		return &em.Status.SyncTypes[pos]
	}

	em.Status.SyncTypes = append(em.Status.SyncTypes, v1alpha1.ClusterSyncType{
		Name:           syncName,
		Message:        message,
		Status:         v1alpha1.Unknown,
		LastUpdateTime: metav1.Now(),
	})

	return &em.Status.SyncTypes[len(em.Status.SyncTypes)-1]
}

func (em *ExecutorSyncTypeManger) syncExecutorPhaseFromCluster(clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) {

	// upgrading, err := masterIsUpgrading(clientSet, tc)
	// if err != nil {
	// 	syncState.Unknown(v1alpha1.UpgradeType, err.Error())
	// }
	//
	// if upgrading {
	// 	syncState.Ongoing(v1alpha1.UpgradeType, "")
	// } else {
	// 	syncState.Complied(v1alpha1.UpgradeType, "")
	// }

}
func (em *ExecutorSyncTypeManger) syncExecutorCreatPhase(clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) {
	panic("implement me")
}

func (em *ExecutorSyncTypeManger) syncExecutorScalePhase(clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) {
	panic("implement me")
}

func (em *ExecutorSyncTypeManger) syncExecutorUpgradePhase(clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) {
	// ns := tc.GetNamespace
	// tcName := tc.GetName()
	// instanceName := tc.GetInstanceName()
	//
	// sts, err := clientSet.AppsV1().StatefulSets(ns).
	// 	Get(context.TODO(), masterMemberName(tcName), metav1.GetOptions{})
	// if err != nil {
	// 	if errors.IsNotFound(err) {
	// 		return false, nil
	// 	} else {
	// 		return false, fmt.Errorf("master [%s/%s] status get satatefulSet error: %v",
	// 			ns, tcName, err)
	// 	}
	// }
	//
	// if isUpgrading(sts) {
	// 	return true, nil
	// }
	//
	// selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	// if err != nil {
	// 	return false, fmt.Errorf("master [%s/%s] condition listing master's pods error: %v",
	// 		ns, instanceName, err)
	// }
	//
	// masterPods, err := clientSet.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{
	// 	LabelSelector: selector.String(),
	// })
	// if err != nil {
	// 	return false, fmt.Errorf("master [%s/%s] condition listing master's pods error: %v",
	// 		ns, instanceName, err)
	// }
	//
	// for _, pod := range masterPods.Items {
	// 	revisionHash, exist := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
	// 	if !exist {
	// 		return false, nil
	// 	}
	// 	if revisionHash != tc.Status.Master.StatefulSet.UpdateRevision {
	// 		return true, nil
	// 	}
	// }
	// return false, nil
}
