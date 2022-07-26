package condition

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/result"
	"github.com/pingcap/tiflow-operator/pkg/status"
	"github.com/pingcap/tiflow-operator/pkg/tiflowapi"
)

type executorConditionManager struct {
	*v1alpha1.TiflowCluster
	cli       client.Client
	clientSet kubernetes.Interface
}

func NewExecutorConditionManager(cli client.Client, clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) ClusterCondition {
	return &executorConditionManager{
		TiflowCluster: tc,
		cli:           cli,
		clientSet:     clientSet,
	}
}

func (ecm *executorConditionManager) Verify(ctx context.Context) error {
	if ecm.Spec.Executor == nil {
		// todo: more gracefully
		SetTrue(v1alpha1.ExecutorVersionChecked, ecm.GetClusterStatus(), metav1.Now())
		SetTrue(v1alpha1.ExecutorReplicaChecked, ecm.GetClusterStatus(), metav1.Now())
		SetTrue(v1alpha1.ExecutorReadyChecked, ecm.GetClusterStatus(), metav1.Now())
		SetTrue(v1alpha1.ExecutorPVCChecked, ecm.GetClusterStatus(), metav1.Now())
		SetTrue(v1alpha1.ExecutorsInfoUpdatedChecked, ecm.GetClusterStatus(), metav1.Now())
		return nil
	}

	ns := ecm.GetNamespace()
	tcName := ecm.GetName()

	sts, err := ecm.verifyStatefulSet(ctx)
	if err != nil {
		return result.NotReadyErr{
			Err: err,
		}
	}

	// todo: need to verify this
	if ecm.pvcVerify() {
		SetTrue(v1alpha1.ExecutorPVCChecked, ecm.GetClusterStatus(), metav1.Now())
	} else {
		SetFalse(v1alpha1.ExecutorPVCChecked, ecm.GetClusterStatus(), metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("executor [%s/%s] verify: pvc's status is abnormal", ns, tcName),
		}
	}

	if ecm.versionVerify() {
		SetTrue(v1alpha1.ExecutorVersionChecked, ecm.GetClusterStatus(), metav1.Now())
	} else {
		SetFalse(v1alpha1.ExecutorVersionChecked, ecm.GetClusterStatus(), metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("executor [%s/%s] verify: version are not up-to-date", ns, tcName),
		}
	}

	// todo: need to handle failureMembers
	if ecm.replicasVerify() {
		SetTrue(v1alpha1.ExecutorReplicaChecked, ecm.GetClusterStatus(), metav1.Now())
	} else {
		SetFalse(v1alpha1.ExecutorReplicaChecked, ecm.GetClusterStatus(), metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("executor [%s/%s] verify: actual is not equal to desired replicas ", ns, tcName),
		}
	}

	if ecm.readyVerify() {
		SetTrue(v1alpha1.ExecutorReadyChecked, ecm.GetClusterStatus(), metav1.Now())
	} else {
		SetFalse(v1alpha1.ExecutorReadyChecked, ecm.GetClusterStatus(), metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("executor [%s/%s] verify: cluster are not reday", ns, tcName),
		}
	}

	if ecm.leaderVerify() {
		SetTrue(v1alpha1.LeaderChecked, ecm.GetClusterStatus(), metav1.Now())
	} else {
		SetFalse(v1alpha1.LeaderChecked, ecm.GetClusterStatus(), metav1.Now())
		return result.SyncStatusErr{
			Err: fmt.Errorf("executor [%s/%s] verify: can not get leader from master cluster", ns, tcName),
		}
	}

	if err = ecm.Update(ctx, sts); err != nil {
		SetFalse(v1alpha1.ExecutorsInfoUpdatedChecked, ecm.GetClusterStatus(), metav1.Now())
		return err
	}
	SetTrue(v1alpha1.ExecutorsInfoUpdatedChecked, ecm.GetClusterStatus(), metav1.Now())

	if ecm.ExecutorStsDesiredReplicas() == ecm.ExecutorActualMembers() {
		SetTrue(v1alpha1.ExecutorMembersChecked, ecm.GetClusterStatus(), metav1.Now())
	} else {
		SetFalse(v1alpha1.ExecutorMembersChecked, ecm.GetClusterStatus(), metav1.Now())
		return result.SyncStatusErr{
			Err: fmt.Errorf("executor [%s/%s] verify: member infos is incomplete", ns, tcName),
		}
	}

	return nil
}

func (ecm *executorConditionManager) verifyStatefulSet(ctx context.Context) (*appsv1.StatefulSet, error) {
	ns := ecm.GetNamespace()
	tcName := ecm.GetName()

	sts, err := ecm.clientSet.AppsV1().StatefulSets(ns).
		Get(ctx, executorMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("executor [%s/%s] verify get satatefulSet error: %v",
			ns, tcName, err)
	}
	ecm.Status.Executor.StatefulSet = sts.Status.DeepCopy()
	return sts, nil
}

func (ecm *executorConditionManager) Update(ctx context.Context, sts *appsv1.StatefulSet) error {
	if err := ecm.update(ctx, sts); err != nil {
		return result.SyncStatusErr{
			Err: err,
		}
	}

	return nil
}

func (ecm *executorConditionManager) update(ctx context.Context, sts *appsv1.StatefulSet) error {
	if err := ecm.syncMembersStatus(ctx, sts); err != nil {
		return err
	}

	// get follows from podName
	ecm.Status.Executor.Image = ""
	if c := findContainerByName(sts, "tiflow-executor"); c != nil {
		ecm.Status.Executor.Image = c.Image
	}

	// todo: Need to get the info of volumes which running container has bound
	// todo: Waiting for discussion
	ecm.Status.Executor.Volumes = nil
	ecm.Status.Executor.LastUpdateTime = metav1.Now()

	return nil
}

func (ecm *executorConditionManager) syncMembersStatus(ctx context.Context, sts *appsv1.StatefulSet) error {
	ns := ecm.GetNamespace()
	tcName := ecm.GetName()

	if ecm.Heterogeneous() && ecm.WithoutLocalMaster() {
		ns = ecm.Spec.Cluster.Namespace
		tcName = ecm.Spec.Cluster.Name
	}

	tiflowClient := tiflowapi.GetMasterClient(ecm.cli, ns, tcName, "", ecm.IsClusterTLSEnabled())

	// get executors info from master
	executorsInfo, err := tiflowClient.GetExecutors()
	if err != nil {
		selector, selectErr := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
		if selectErr != nil {
			return fmt.Errorf("executor [%s/%s] codition converting statefulset selector error: %v",
				ns, tcName, selectErr)
		}

		// get endpoints info
		eps, epErr := ecm.clientSet.CoreV1().Endpoints(ns).
			List(ctx, metav1.ListOptions{
				LabelSelector: selector.String(),
			})
		if epErr != nil {
			return fmt.Errorf("executor [%s/%s] codition failed to get endpoints %s , err: %s, epErr %s",
				ns, tcName, executorMemberName(tcName), err, epErr)
		}

		// tiflow-executor service has no endpoints
		if eps != nil && eps.Items != nil && len(eps.Items[0].Subsets) == 0 {
			return fmt.Errorf("%s, service %s/%s has no endpoints", err, ns, executorMemberName(tcName))
		}

		return err
	}

	return ecm.updateMembersInfo(executorsInfo)
}

func (ecm *executorConditionManager) updateMembersInfo(executorsInfo tiflowapi.ExecutorsInfo) error {
	ns := ecm.GetNamespace()

	// todo: WIP, get information about the FailureMembers and FailoverUID through the MasterClient
	members := make(map[string]v1alpha1.ExecutorMember)
	peerMembers := make(map[string]v1alpha1.ExecutorMember)
	for _, e := range executorsInfo.Executors {
		member := v1alpha1.ExecutorMember{
			Id:                 e.ID,
			Name:               e.Name,
			Addr:               e.Address,
			Labels:             e.Labels,
			LastTransitionTime: metav1.Now(),
		}

		clusterName, ordinal, namespace, err2 := getOrdinalFromName(e.Name, v1alpha1.TiFlowExecutorMemberType)
		if err2 == nil && clusterName == ecm.GetName() && namespace == ns && ordinal < ecm.ExecutorStsDesiredReplicas() {
			members[e.Name] = member
		} else {
			peerMembers[e.Name] = member
		}
	}

	ecm.Status.Executor.Members = members
	ecm.Status.Executor.PeerMembers = peerMembers
	return nil
}

func (ecm *executorConditionManager) pvcVerify() bool {
	// todo: need to check pvc's status here
	return true
}

func (ecm *executorConditionManager) versionVerify() bool {
	klog.Infof("Executor: CurrentRevision: %s , UpdateRevision: %s",
		ecm.GetExecutorStatus().StatefulSet.CurrentRevision,
		ecm.GetExecutorStatus().StatefulSet.UpdateRevision)

	if status.GetSyncStatus(v1alpha1.UpgradeType, ecm.GetClusterStatus(),
		v1alpha1.TiFlowExecutorMemberType) == v1alpha1.Ongoing {
		return true
	}
	return statefulSetUpToDate(ecm.Status.Executor.StatefulSet, true)
}

func (ecm *executorConditionManager) replicasVerify() bool {
	klog.Infof("DesiredReplicas: %d , CurrentReplicas: %d, UpdatedReplicas: %d",
		ecm.ExecutorStsDesiredReplicas(), ecm.ExecutorStsCurrentReplicas(), ecm.ExecutorStsUpdatedReplicas())

	if statefulSetUpToDate(ecm.Status.Executor.StatefulSet, true) {
		return ecm.ExecutorStsDesiredReplicas() == ecm.ExecutorStsCurrentReplicas()
	}

	return ecm.ExecutorStsDesiredReplicas() == ecm.ExecutorStsCurrentReplicas()+ecm.ExecutorStsUpdatedReplicas()
}

func (ecm *executorConditionManager) readyVerify() bool {
	klog.Infof("DesiredReplicas: %d , ReadyReplicas: %d",
		ecm.ExecutorStsDesiredReplicas(), ecm.ExecutorStsReadyReplicas())

	return ecm.ExecutorStsDesiredReplicas() == ecm.ExecutorStsReadyReplicas()
}

func (ecm *executorConditionManager) leaderVerify() bool {
	ns := ecm.GetNamespace()
	tcName := ecm.GetName()

	if ecm.Heterogeneous() && ecm.WithoutLocalMaster() {
		ns = ecm.Spec.Cluster.Namespace
		tcName = ecm.Spec.Cluster.Name
	}

	tiflowClient := tiflowapi.GetMasterClient(ecm.cli, ns, tcName, "", ecm.IsClusterTLSEnabled())
	leader, err := tiflowClient.GetLeader()
	if err != nil {
		return false
	}

	ecm.Status.Master.Leader = v1alpha1.MasterMember{
		ClientURL:          leader.AdvertiseAddr,
		LastTransitionTime: metav1.Now(),
	}

	return true
}
