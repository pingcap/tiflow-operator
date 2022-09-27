package condition

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/result"
	"github.com/pingcap/tiflow-operator/pkg/tiflowapi"
)

type MasterConditionManager struct {
	cli       client.Client
	clientSet kubernetes.Interface
	cluster   *v1alpha1.TiflowCluster
}

func NewMasterConditionManager(cli client.Client, clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) ClusterCondition {
	return &MasterConditionManager{
		cli:       cli,
		clientSet: clientSet,
		cluster:   tc,
	}
}

func (mcm *MasterConditionManager) Update(ctx context.Context) error {
	if mcm.cluster.Heterogeneous() {
		SetTrue(v1alpha1.MastersInfoUpdatedChecked, &mcm.cluster.Status, metav1.Now())
		return nil
	}

	SetFalse(v1alpha1.MasterSynced, &mcm.cluster.Status, metav1.Now())

	ns := mcm.cluster.GetNamespace()
	tcName := mcm.cluster.GetName()

	sts, err := mcm.clientSet.AppsV1().StatefulSets(ns).
		Get(ctx, masterMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		} else {
			return fmt.Errorf("master [%s/%s] codition get satatefulSet error: %v",
				ns, tcName, err)
		}
	}

	tiflowClient := tiflowapi.GetMasterClient(mcm.cli, ns, tcName, "", mcm.cluster.IsClusterTLSEnabled())
	mastersInfo, err := tiflowClient.GetMasters()
	if err != nil {
		SetFalse(v1alpha1.MastersInfoUpdatedChecked, &mcm.cluster.Status, metav1.Now())
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

	leader, err := tiflowClient.GetLeader()
	if err != nil {
		return err
	}

	mcm.cluster.Status.Master.Leader = v1alpha1.MasterMember{
		ClientURL: leader.AdvertiseAddr,
		Health:    true,
	}

	mcm.cluster.Status.Master.Image = ""
	c := findContainerByName(sts, "tiflow-master")
	if c != nil {
		mcm.cluster.Status.Master.Image = c.Image
	}

	SetTrue(v1alpha1.MastersInfoUpdatedChecked, &mcm.cluster.Status, metav1.Now())
	return nil
}

func (mcm *MasterConditionManager) Check() error {
	if mcm.cluster.Heterogeneous() {
		// todo: more gracefully
		SetTrue(v1alpha1.MasterNumChecked, &mcm.cluster.Status, metav1.Now())
		SetTrue(v1alpha1.MasterReadyChecked, &mcm.cluster.Status, metav1.Now())
		return nil
	}

	ns := mcm.cluster.GetNamespace()
	tcName := mcm.cluster.GetName()

	infosUpdateChecked := True(v1alpha1.MastersInfoUpdatedChecked, mcm.cluster.Status.ClusterConditions)
	if !infosUpdateChecked {
		return result.SyncConditionErr{
			Err: fmt.Errorf("master [%s/%s] check: information update failed", ns, tcName),
		}
	}

	actual := int32(len(mcm.cluster.Status.Master.Members))
	desired := mcm.cluster.MasterStsDesiredReplicas()
	// todo: need to handle failureMembers
	// failed := len(mcm.cluster.Status.Master.FailureMembers)

	if actual == desired {
		SetTrue(v1alpha1.MasterNumChecked, &mcm.cluster.Status, metav1.Now())
	} else {
		SetFalse(v1alpha1.MasterNumChecked, &mcm.cluster.Status, metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("master [%s/%s] check: actual is not equal to desired replicas ", ns, tcName),
		}
	}

	if mcm.cluster.AllMasterMembersReady() {
		SetTrue(v1alpha1.MasterReadyChecked, &mcm.cluster.Status, metav1.Now())
	} else {
		SetFalse(v1alpha1.MasterReadyChecked, &mcm.cluster.Status, metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("master [%s/%s] check: master not reday", ns, tcName),
		}
	}

	return nil
}

func (mcm *MasterConditionManager) updateMembersInfo(mastersInfo tiflowapi.MastersInfo) error {
	ns := mcm.cluster.GetNamespace()

	// TODO: WIP, need to get the information of memberDeleted and LastTransitionTime
	members := make(map[string]v1alpha1.MasterMember)
	for _, m := range mastersInfo.Masters {
		// TODO: WIP
		if !strings.Contains(m.Address, ns) {
			continue
		}

		masterName, err := formatName(m.Address)
		if err != nil {
			return err
		}

		members[masterName] = v1alpha1.MasterMember{
			Id:                 m.ID,
			Address:            m.Address,
			IsLeader:           m.IsLeader,
			PodName:            m.Name,
			Health:             true,
			LastTransitionTime: metav1.Now(),
		}
	}

	mcm.cluster.Status.Master.Members = members
	return nil
}

// func (mcm *MasterConditionManager) statefulSetIsUpgrading(ctx context.Context, sts *appsv1.StatefulSet) (bool, error) {
// 	if mngerutils.StatefulSetIsUpgrading(sts) {
// 		return true, nil
// 	}
//
// 	ns := mcm.cluster.GetNamespace()
// 	instanceName := mcm.cluster.GetInstanceName()
// 	selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
// 	if err != nil {
// 		return false, fmt.Errorf("master [%s/%s] condition converting selector error: %v",
// 			ns, instanceName, err)
// 	}
// 	masterPods, err := mcm.clientSet.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
// 		LabelSelector: selector.String(),
// 	})
// 	if err != nil {
// 		return false, fmt.Errorf("master [%s/%s] condition listing master's pods error: %v",
// 			ns, instanceName, err)
// 	}
//
// 	for _, pod := range masterPods.Items {
// 		revisionHash, exist := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
// 		if !exist {
// 			return false, nil
// 		}
// 		if revisionHash != mcm.cluster.Status.Master.StatefulSet.UpdateRevision {
// 			return true, nil
// 		}
// 	}
// 	return false, nil
// }