package member

import (
	"fmt"

	"github.com/pingcap/errors"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/controller"
	"github.com/pingcap/tiflow-operator/pkg/tiflowapi"
)

type masterScaler struct {
	cli client.Client
}

// NewMasterScaler returns a DMScaler
func NewMasterScaler(cli client.Client) Scaler {
	return &masterScaler{
		cli: cli,
	}
}

func (s masterScaler) Scale(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error {
	scaling := *desired.Spec.Replicas - *actual.Spec.Replicas
	if scaling > 0 {
		return s.ScaleOut(meta, actual, desired)
	} else if scaling < 0 {
		return s.ScaleIn(meta, actual, desired)
	}
	return nil
}

func (s masterScaler) ScaleOut(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error {
	tc, ok := meta.(*v1alpha1.TiflowCluster)
	if !ok {
		return nil
	}

	klog.Infof("scaling out tiflow-master statefulset %s/%s, ordinal: %d, oldReplicas: %d, newReplicas: %d)", actual.Namespace, actual.Name, *actual.Spec.Replicas, *actual.Spec.Replicas, *desired.Spec.Replicas)

	if !tc.Status.Master.Synced {
		return errors.Errorf("tiflow cluster: %s/%s's tiflow-master status sync failed, can't scale out now", tc.GetNamespace(), tc.GetName())
	}
	*desired.Spec.Replicas = *actual.Spec.Replicas + 1
	return nil
}

func (s masterScaler) ScaleIn(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error {
	tc, ok := meta.(*v1alpha1.TiflowCluster)
	if !ok {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if !tc.Status.Master.Synced {
		return errors.Errorf("tiflow cluster: %s/%s's tiflow-master status sync failed, can't scale out now", ns, tcName)
	}

	ordinal := *actual.Spec.Replicas - 1
	memberName := ordinalPodName(v1alpha1.TiFlowMasterMemberType, tcName, ordinal)

	if !tc.Status.Master.Synced {
		return fmt.Errorf("tiflow cluster: %s/%s's tiflow-master status sync failed, can't scale in now", ns, tcName)
	}

	klog.Infof("scaling in tiflow-master statefulset %s/%s, ordinal: %d, oldReplicas: %d, newReplicas: %d", actual.Namespace, actual.Name, ordinal, *actual.Spec.Replicas, *desired.Spec.Replicas)
	// If the tiflow-master pod was tiflow-master leader during scale-in, we would evict tiflow-master leader first
	// If it's the last member we don't need to do this because we will delete this later
	if ordinal > 0 {
		if tc.Status.Master.Leader.ClientURL == memberName {
			masterPeerClient := tiflowapi.GetMasterClient(s.cli, ns, tcName, memberName, tc.Spec.TLSCluster)
			err := masterPeerClient.EvictLeader()
			if err != nil {
				return err
			}
			return controller.RequeueErrorf("tiflow cluster [%s/%s]'s tiflow-master pod[%s/%s] is transferring tiflow-master leader, can't scale-in now", ns, tcName, ns, memberName)
		}
	}

	*desired.Spec.Replicas = *actual.Spec.Replicas - 1
	return nil
}
