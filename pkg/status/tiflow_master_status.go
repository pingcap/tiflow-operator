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

type masterPhaseManager struct {
	*v1alpha1.TiflowCluster
	cli       client.Client
	clientSet kubernetes.Interface
}

func NewMasterPhaseManager(cli client.Client, clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) SyncPhaseManager {
	return &masterPhaseManager{
		tc,
		cli,
		clientSet,
	}
}

func setMasterClusterStatusOnFirstReconcile(masterStatus *v1alpha1.MasterStatus) {
	InitMasterClusterSyncTypesIfNeed(masterStatus)
	if masterStatus.Phase != "" {
		return
	}

	masterStatus.Phase = v1alpha1.MasterStarting
	masterStatus.Message = "Starting... tiflow-master on first reconcile. Just a moment"
	masterStatus.LastTransitionTime = metav1.Now()
	masterStatus.LastUpdateTime = metav1.Now()
	return
}

func InitMasterClusterSyncTypesIfNeed(masterStatus *v1alpha1.MasterStatus) {
	if masterStatus.SyncTypes == nil {
		masterStatus.SyncTypes = []v1alpha1.ClusterSyncType{}
	}
	return
}

// SyncPhase
// todo: need to check for all OperatorActions or for just the 0th element
// This depends on our logic for updating Status
func (mm *masterPhaseManager) SyncPhase() {
	masterStatus := mm.GetMasterStatus()
	InitMasterClusterSyncTypesIfNeed(masterStatus)

	if !mm.syncMasterPhaseFromCluster() {
		return
	}

	for _, sync := range masterStatus.SyncTypes {
		if sync.Status == v1alpha1.Completed {
			continue
		}

		switch sync.Status {
		case v1alpha1.Failed:
			masterStatus.Phase = v1alpha1.MasterFailed
		case v1alpha1.Unknown:
			masterStatus.Phase = v1alpha1.MasterUnknown
		default:
			masterStatus.Phase = sync.Name.GetMasterClusterPhase()
		}

		masterStatus.Message = sync.Message
		masterStatus.LastTransitionTime = metav1.Now()
		return
	}

	masterStatus.Phase = v1alpha1.MasterRunning
	masterStatus.Message = "Ready..., tiflow-master reconcile completed successfully. Enjoying..."
	masterStatus.LastTransitionTime = metav1.Now()
	return
}

func (mm *masterPhaseManager) syncMasterPhaseFromCluster() bool {

	if mm.syncMasterCreatePhase() {
		return true
	}

	if mm.syncMasterScalePhase() {
		return true
	}

	if mm.syncMasterUpgradePhase() {
		return true
	}

	return false
}
func (mm *masterPhaseManager) syncMasterCreatePhase() bool {
	syncTypes := mm.GetMasterSyncTypes()
	index := findPos(v1alpha1.CreateType, syncTypes)
	if index < 0 || syncTypes[index].Status == v1alpha1.Completed {
		return false
	}

	if mm.MasterStsDesiredReplicas() == mm.MasterStsCurrentReplicas() {
		Completed(v1alpha1.CreateType, mm.GetClusterStatus(), v1alpha1.TiFlowMasterMemberType, "master creating completed")
		return true
	}

	return false
}

func (mm *masterPhaseManager) syncMasterScalePhase() bool {
	syncTypes := mm.GetMasterSyncTypes()

	index := findPos(v1alpha1.ScaleOutType, syncTypes)
	if index >= 0 && syncTypes[index].Status != v1alpha1.Completed {
		if mm.MasterStsDesiredReplicas() == mm.MasterStsCurrentReplicas() {
			Completed(v1alpha1.ScaleOutType, mm.GetClusterStatus(), v1alpha1.TiFlowMasterMemberType, "master scaling out completed")
			return true
		}
		return false
	}

	index = findPos(v1alpha1.ScaleInType, syncTypes)
	if index >= 0 && syncTypes[index].Status != v1alpha1.Completed {
		if mm.MasterStsDesiredReplicas() == mm.MasterStsCurrentReplicas() {
			Completed(v1alpha1.ScaleInType, mm.GetClusterStatus(), v1alpha1.TiFlowMasterMemberType, "master scaling in completed")
			return true
		}
		return false
	}

	return false
}

// syncMasterUpgradePhase return true indicates a change in the current phase
func (mm *masterPhaseManager) syncMasterUpgradePhase() bool {
	syncTypes := mm.GetMasterSyncTypes()

	index := findPos(v1alpha1.UpgradeType, syncTypes)

	if index < 0 || syncTypes[index].Status == v1alpha1.Completed {
		return false
	}

	ns := mm.GetNamespace()
	tcName := mm.GetName()

	sts, err := mm.clientSet.AppsV1().StatefulSets(ns).
		Get(context.TODO(), masterMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("master [%s/%s] upgrade phase: can not get statefulSet", ns, tcName)
		Unknown(v1alpha1.UpgradeType, mm.GetClusterStatus(), v1alpha1.TiFlowMasterMemberType, message)
		return true
	}

	if isUpgrading(sts) {
		return false
	}

	instanceName := mm.GetInstanceName()
	selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	if err != nil {
		message := fmt.Sprintf("master [%s/%s] upgrade phase: converting statefulSet selector error: %v",
			ns, instanceName, err)
		Failed(v1alpha1.UpgradeType, mm.GetClusterStatus(), v1alpha1.TiFlowMasterMemberType, message)
		return true
	}

	masterPods, err := mm.clientSet.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		message := fmt.Sprintf("master [%s/%s] upgrade phase: listing master's pods error: %v",
			ns, instanceName, err)
		Failed(v1alpha1.UpgradeType, mm.GetClusterStatus(), v1alpha1.TiFlowMasterMemberType, message)
		return true
	}

	// todo: more gracefully
	for _, pod := range masterPods.Items {
		revisionHash, exist := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
		if !exist {
			message := fmt.Sprintf("master [%s/%s] upgrade phase: has no label ControllerRevisionHashLabelKey",
				ns, tcName)
			Failed(v1alpha1.UpgradeType, mm.GetClusterStatus(), v1alpha1.TiFlowMasterMemberType, message)
			return true
		}
		if revisionHash != mm.GetMasterStatus().StatefulSet.UpdateRevision {
			return false
		}
	}
	Completed(v1alpha1.UpgradeType, mm.GetClusterStatus(), v1alpha1.TiFlowMasterMemberType, "master upgrading  completed")
	return true
}
