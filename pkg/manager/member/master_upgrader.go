package member

import (
	"context"
	"fmt"
	"strings"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/condition"
	"github.com/pingcap/tiflow-operator/pkg/controller"
	mngerutils "github.com/pingcap/tiflow-operator/pkg/manager/utils"
	"github.com/pingcap/tiflow-operator/pkg/status"
	"github.com/pingcap/tiflow-operator/pkg/tiflowapi"
)

type masterUpgrader struct {
	cli client.Client
}

// NewMasterUpgrader returns a masterUpgrader
func NewMasterUpgrader(cli client.Client) Upgrader {
	return &masterUpgrader{
		cli: cli,
	}
}

func (u *masterUpgrader) Upgrade(tc *v1alpha1.TiflowCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	return u.gracefulUpgrade(tc, oldSet, newSet)
}

func (u *masterUpgrader) gracefulUpgrade(tc *v1alpha1.TiflowCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	condition.SetFalse(v1alpha1.MasterSynced, &tc.Status, metav1.Now())
	syncState := status.NewMasterSyncTypeManager(&tc.Status.Master)
	syncState.Ongoing(v1alpha1.UpgradeType,
		fmt.Sprintf("tiflow master [%s/%s] upgrading...", ns, tcName))

	if tc.MasterScaling() {
		klog.Infof("TiflowCluster: [%s/%s]'s tiflow-master is scaling, can not upgrade tiflow-master", ns, tcName)
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if tc.Status.Master.StatefulSet.UpdateRevision == tc.Status.Master.StatefulSet.CurrentRevision {
		return nil
	}

	if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
		// Manually bypass tiflow-operator to modify statefulset directly, such as modify tiflow-master statefulset's RollingUpdate straregy to OnDelete strategy,
		// or set RollingUpdate to nil, skip tiflow-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
		// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading tiflow-master.
		// Therefore, in the production environment, we should try to avoid modifying the tiflow-master statefulset update strategy directly.
		newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
		klog.Warningf("tiflowcluster: [%s/%s] tiflow-master statefulset %s UpdateStrategy has been modified manually", ns, tcName, oldSet.GetName())
		return nil
	}

	mngerutils.SetUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	for i := *oldSet.Spec.Replicas - 1; i >= 0; i-- {
		podName := TiflowMasterPodName(tcName, i)
		pod := &v1.Pod{}
		err := u.cli.Get(context.TODO(), types.NamespacedName{
			Namespace: ns,
			Name:      podName,
		}, pod)
		if err != nil {
			return fmt.Errorf("gracefulUpgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}

		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tiflowcluster: [%s/%s]'s tiflow-master pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.Master.StatefulSet.UpdateRevision {
			if !podutil.IsPodReady(pod) {
				return controller.RequeueErrorf("tiflowcluster: [%s/%s]'s upgraded tiflow pod: [%s] is not ready", ns, tcName, podName)
			}
			// TODO: add this after members update is supported
			// if member, exist := tc.Status.Master.Members[podName]; !exist || !member.Health {
			//	return controller.RequeueErrorf("tiflowcluster: [%s/%s]'s tiflow-master upgraded pod: [%s] is not ready", ns, tcName, podName)
			// }
			continue
		}

		return u.upgradeMasterPod(tc, i, newSet)
	}

	return nil
}

func (u *masterUpgrader) upgradeMasterPod(tc *v1alpha1.TiflowCluster, ordinal int32, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	upgradePodName := TiflowMasterPodName(tcName, ordinal)
	if strings.Contains(tc.Status.Master.Leader.ClientURL, TiflowMasterPeerSvcName(tcName, ordinal)) && tc.MasterStsActualReplicas() > 1 {
		err := u.evictMasterLeader(tc, upgradePodName)
		if err != nil {
			klog.Errorf("tiflow-master upgrader: failed to evict tiflow-master %s's leader: %v", upgradePodName, err)
			return err
		}
		klog.Infof("tiflow-master upgrader: evict tiflow-master %s's leader successfully", upgradePodName)
		return controller.RequeueErrorf("tiflowcluster: [%s/%s]'s tiflow-master member: evicting [%s]'s leader", ns, tcName, upgradePodName)
	}

	mngerutils.SetUpgradePartition(newSet, ordinal)
	return nil
}

func (u *masterUpgrader) evictMasterLeader(tc *v1alpha1.TiflowCluster, podName string) error {
	return tiflowapi.GetMasterClient(u.cli, tc.GetNamespace(), tc.GetName(), podName, tc.IsClusterTLSEnabled()).EvictLeader()
}
