package condition

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/result"
)

type TiflowClusterConditionManager struct {
	masterCondition   ClusterCondition
	executorCondition ClusterCondition
	cluster           *v1alpha1.TiflowCluster
}

func NewTiflowCLusterConditionManager(cli client.Client, clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) Condition {
	return &TiflowClusterConditionManager{
		masterCondition:   NewMasterConditionManager(cli, clientSet, tc),
		executorCondition: NewExecutorConditionManager(cli, clientSet, tc),
		cluster:           tc,
	}
}

func (tcm *TiflowClusterConditionManager) Sync(ctx context.Context) error {

	if err := tcm.masterCondition.Update(ctx); err != nil {
		return result.UpdateClusterStatus{
			Err: err,
		}
	}

	if err := tcm.executorCondition.Update(ctx); err != nil {
		return result.UpdateClusterStatus{
			Err: err,
		}
	}

	return tcm.Check(ctx)
}

func (tcm *TiflowClusterConditionManager) Check(ctx context.Context) error {
	if err := tcm.masterCondition.Check(ctx); err != nil {
		return err
	}

	if err := tcm.executorCondition.Check(ctx); err != nil {
		return err
	}

	return tcm.Apply(ctx)
}

func (tcm *TiflowClusterConditionManager) Apply(ctx context.Context) error {
	tcm.cluster.Status.Master.Synced = true
	SetTrue(v1alpha1.MasterSynced, &tcm.cluster.Status, metav1.Now())

	tcm.cluster.Status.Executor.Synced = true
	SetTrue(v1alpha1.ExecutorSynced, &tcm.cluster.Status, metav1.Now())

	SetTrue(v1alpha1.VersionChecked, &tcm.cluster.Status, metav1.Now())
	SetTrue(v1alpha1.InitializedCondition, &tcm.cluster.Status, metav1.Now())

	return nil
}
