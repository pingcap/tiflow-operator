package condition

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
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
	InitConditionsIfNeed(&tcm.cluster.Status, metav1.Now())

	if err := tcm.masterCondition.Update(ctx); err != nil {
		return err
	}

	if err := tcm.executorCondition.Update(ctx); err != nil {
		return err
	}

	return tcm.Apply()
}

func (tcm *TiflowClusterConditionManager) Apply() error {

	if err := tcm.masterCondition.Check(); err != nil {
		return err
	}

	if err := tcm.executorCondition.Check(); err != nil {
		return err
	}

	SetTrue(v1alpha1.VersionChecked, &tcm.cluster.Status, metav1.Now())
	SetTrue(v1alpha1.MasterSynced, &tcm.cluster.Status, metav1.Now())
	SetTrue(v1alpha1.ExecutorSynced, &tcm.cluster.Status, metav1.Now())

	return nil
}
