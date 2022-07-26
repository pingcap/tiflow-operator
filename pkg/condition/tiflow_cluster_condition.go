package condition

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/tiflowapi"
)

type TiflowClusterConditionManager struct {
	*v1alpha1.TiflowCluster
	cli               client.Client
	masterCondition   ClusterCondition
	executorCondition ClusterCondition
}

func NewTiflowClusterConditionManager(cli client.Client, clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) Condition {
	return &TiflowClusterConditionManager{
		TiflowCluster:     tc,
		cli:               cli,
		masterCondition:   NewMasterConditionManager(cli, clientSet, tc),
		executorCondition: NewExecutorConditionManager(cli, clientSet, tc),
	}
}

func (tcm *TiflowClusterConditionManager) Sync(ctx context.Context) error {
	InitConditionsIfNeed(tcm.GetClusterStatus(), metav1.Now())

	if err := tcm.masterCondition.Verify(ctx); err != nil {
		return err
	}

	if err := tcm.executorCondition.Verify(ctx); err != nil {
		return err
	}

	return tcm.Apply()
}

func (tcm *TiflowClusterConditionManager) Apply() error {
	tcm.GetClusterStatus().Master.ServerAddress = tiflowapi.GetMasterClient(tcm.cli,
		tcm.GetNamespace(), tcm.GetName(), "", tcm.IsClusterTLSEnabled()).
		GetURL()

	SetTrue(v1alpha1.MasterSyncChecked, tcm.GetClusterStatus(), metav1.Now())
	SetTrue(v1alpha1.ExecutorSyncChecked, tcm.GetClusterStatus(), metav1.Now())
	SetTrue(v1alpha1.VersionChecked, tcm.GetClusterStatus(), metav1.Now())
	return nil
}
