package condition

import (
	"context"
	"fmt"
	"github.com/pingcap/tiflow-operator/pkg/status"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/klog/v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/result"
	"github.com/pingcap/tiflow-operator/pkg/tiflowapi"
)

type masterConditionManager struct {
	*v1alpha1.TiflowCluster
	cli       client.Client
	clientSet kubernetes.Interface
}

func NewMasterConditionManager(cli client.Client, clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) ClusterCondition {
	return &masterConditionManager{
		TiflowCluster: tc,
		cli:           cli,
		clientSet:     clientSet,
	}
}

func (mcm *masterConditionManager) Verify(ctx context.Context) error {
	if mcm.Heterogeneous() {
		// todo: more gracefully
		SetTrue(v1alpha1.MasterVersionChecked, mcm.GetClusterStatus(), metav1.Now())
		SetTrue(v1alpha1.MasterReplicaChecked, mcm.GetClusterStatus(), metav1.Now())
		SetTrue(v1alpha1.MasterReadyChecked, mcm.GetClusterStatus(), metav1.Now())
		SetTrue(v1alpha1.LeaderChecked, mcm.GetClusterStatus(), metav1.Now())
		SetTrue(v1alpha1.MastersInfoUpdatedChecked, mcm.GetClusterStatus(), metav1.Now())
		return nil
	}

	ns := mcm.GetNamespace()
	tcName := mcm.GetName()

	sts, err := mcm.verifyStatefulSet(ctx)
	if err != nil {
		return result.NotReadyErr{
			Err: err,
		}
	}

	if mcm.versionVerify() {
		SetTrue(v1alpha1.MasterVersionChecked, mcm.GetClusterStatus(), metav1.Now())
	} else {
		SetFalse(v1alpha1.MasterVersionChecked, mcm.GetClusterStatus(), metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("master [%s/%s] verify: version are not up-to-date", ns, tcName),
		}
	}

	// todo: need to handle failureMembers
	if mcm.replicasVerify() {
		SetTrue(v1alpha1.MasterReplicaChecked, mcm.GetClusterStatus(), metav1.Now())
	} else {
		SetFalse(v1alpha1.MasterReplicaChecked, mcm.GetClusterStatus(), metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("master [%s/%s] verify: actual is not equal to desired replicas ", ns, tcName),
		}
	}

	if mcm.readyVerify() {
		SetTrue(v1alpha1.MasterReadyChecked, mcm.GetClusterStatus(), metav1.Now())
	} else {
		SetFalse(v1alpha1.MasterReadyChecked, mcm.GetClusterStatus(), metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("master [%s/%s] verify: cluster are not reday", ns, tcName),
		}
	}

	if mcm.leaderVerify() {
		SetTrue(v1alpha1.LeaderChecked, mcm.GetClusterStatus(), metav1.Now())
	} else {
		SetFalse(v1alpha1.LeaderChecked, mcm.GetClusterStatus(), metav1.Now())
		return result.SyncStatusErr{
			Err: fmt.Errorf("master [%s/%s] verify: can not get leader from master cluster", ns, tcName),
		}
	}

	if err = mcm.Update(ctx, sts); err != nil {
		SetFalse(v1alpha1.MastersInfoUpdatedChecked, mcm.GetClusterStatus(), metav1.Now())
		return err
	}
	SetTrue(v1alpha1.MastersInfoUpdatedChecked, mcm.GetClusterStatus(), metav1.Now())

	// todo: unclear behavior of peerMembers
	if mcm.MasterStsDesiredReplicas() == mcm.MasterActualMembers() {
		SetTrue(v1alpha1.MasterMembersChecked, mcm.GetClusterStatus(), metav1.Now())
	} else {
		return result.SyncStatusErr{
			Err: fmt.Errorf("master [%s/%s] verify: member infos is incomplete", ns, tcName),
		}
	}

	return nil
}

func (mcm *masterConditionManager) verifyStatefulSet(ctx context.Context) (*appsv1.StatefulSet, error) {
	ns := mcm.GetNamespace()
	tcName := mcm.GetName()

	sts, err := mcm.clientSet.AppsV1().StatefulSets(ns).
		Get(ctx, masterMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("master [%s/%s] verify get satatefulSet error: %v",
			ns, tcName, err)
	}
	mcm.Status.Master.StatefulSet = sts.Status.DeepCopy()
	return sts, nil
}

func (mcm *masterConditionManager) Update(ctx context.Context, sts *appsv1.StatefulSet) error {
	if err := mcm.update(ctx, sts); err != nil {
		return result.SyncStatusErr{
			Err: err,
		}
	}

	return nil
}

func (mcm *masterConditionManager) update(ctx context.Context, sts *appsv1.StatefulSet) error {
	ns := mcm.GetNamespace()
	tcName := mcm.GetName()

	tiflowClient := tiflowapi.GetMasterClient(mcm.cli, ns, tcName, "", mcm.IsClusterTLSEnabled())

	mastersInfo, err := tiflowClient.GetMasters()
	if err != nil {
		selector, selectErr := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
		if selectErr != nil {
			return fmt.Errorf("master [%s/%s] codition converting statefulset selector error: %v",
				ns, tcName, selectErr)
		}

		// get endpoints info
		eps, epErr := mcm.clientSet.CoreV1().Endpoints(ns).
			List(ctx, metav1.ListOptions{
				LabelSelector: selector.String(),
			})
		if epErr != nil {
			return fmt.Errorf("master [%s/%s] codition failed to get endpoints %s , err: %s, epErr %s",
				ns, tcName, masterMemberName(tcName), err, epErr)
		}

		// tiflow-master service has no endpoints
		if eps != nil && eps.Items != nil && len(eps.Items[0].Subsets) == 0 {
			return fmt.Errorf("%s, service %s/%s has no endpoints", err, ns, masterMemberName(tcName))
		}

		return err
	}

	if err = mcm.updateMembersInfo(mastersInfo); err != nil {
		return err
	}

	mcm.Status.Master.Image = ""
	c := findContainerByName(sts, "tiflow-master")
	if c != nil {
		mcm.Status.Master.Image = c.Image
	}

	mcm.Status.Master.LastUpdateTime = metav1.Now()

	return nil
}

func (mcm *masterConditionManager) updateMembersInfo(mastersInfo tiflowapi.MastersInfo) error {
	ns := mcm.GetNamespace()

	// TODO: WIP, need to get the information of memberDeleted and LastTransitionTime
	members := make(map[string]v1alpha1.MasterMember)
	peerMembers := make(map[string]v1alpha1.MasterMember)
	for _, m := range mastersInfo.Masters {
		member := v1alpha1.MasterMember{
			Id:                 m.ID,
			Address:            m.Address,
			IsLeader:           m.IsLeader,
			Name:               m.Name,
			LastTransitionTime: metav1.Now(),
		}
		clusterName, ordinal, namespace, err2 := getOrdinalFromName(m.Name, v1alpha1.TiFlowMasterMemberType)
		if err2 == nil && clusterName == mcm.GetName() && namespace == ns && ordinal < mcm.MasterStsDesiredReplicas() {
			members[m.Name] = member
		} else {
			peerMembers[m.Name] = member
		}
	}

	mcm.Status.Master.Members = members
	mcm.Status.Master.PeerMembers = peerMembers
	return nil
}

func (mcm *masterConditionManager) versionVerify() bool {
	klog.Infof("Master: CurrentRevision: %d , UpdateRevision: %d",
		mcm.GetMasterStatus().StatefulSet.CurrentRevision,
		mcm.GetMasterStatus().StatefulSet.UpdateRevision)

	if status.GetSyncStatus(v1alpha1.UpgradeType, mcm.GetClusterStatus(),
		v1alpha1.TiFlowMasterMemberType) == v1alpha1.Ongoing {
		return true
	}

	return statefulSetUpToDate(mcm.Status.Master.StatefulSet, true)
}

func (mcm *masterConditionManager) replicasVerify() bool {
	klog.Infof("DesiredReplicas: %d , CurrentReplicas: %d, UpdatedReplicas: %d",
		mcm.MasterStsDesiredReplicas(), mcm.MasterStsCurrentReplicas(), mcm.MasterStsUpdatedReplicas())

	if statefulSetUpToDate(mcm.Status.Master.StatefulSet, true) {
		return mcm.MasterStsDesiredReplicas() == mcm.MasterStsCurrentReplicas()
	}

	return mcm.MasterStsDesiredReplicas() == mcm.MasterStsCurrentReplicas()+mcm.MasterStsUpdatedReplicas()

}

func (mcm *masterConditionManager) readyVerify() bool {
	klog.Infof("DesiredReplicas: %d , ReadyReplicas: %d",
		mcm.MasterStsDesiredReplicas(), mcm.MasterStsReadyReplicas())

	return mcm.MasterStsDesiredReplicas() == mcm.MasterStsReadyReplicas()
}

func (mcm *masterConditionManager) leaderVerify() bool {
	ns := mcm.GetNamespace()
	tcName := mcm.GetName()

	tiflowClient := tiflowapi.GetMasterClient(mcm.cli, ns, tcName, "", mcm.IsClusterTLSEnabled())
	leader, err := tiflowClient.GetLeader()
	if err != nil {
		return false
	}

	mcm.Status.Master.Leader = v1alpha1.MasterMember{
		ClientURL:          leader.AdvertiseAddr,
		LastTransitionTime: metav1.Now(),
	}

	return true
}
