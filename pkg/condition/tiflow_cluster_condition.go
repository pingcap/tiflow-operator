package condition

import (
	"context"
	"github.com/pingcap/tiflow-operator/pkg/tiflowapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

type TiflowClusterConditionManager struct {
	*v1alpha1.TiflowCluster
	cli               client.Client
	masterCondition   ClusterCondition
	executorCondition ClusterCondition
}

func NewTiflowCLusterConditionManager(cli client.Client, clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) Condition {
	return &TiflowClusterConditionManager{
		TiflowCluster:     tc,
		cli:               cli,
		masterCondition:   NewMasterConditionManager(cli, clientSet, tc),
		executorCondition: NewExecutorConditionManager(cli, clientSet, tc),
	}
}

func (tcm *TiflowClusterConditionManager) Sync(ctx context.Context) error {
	InitConditionsIfNeed(tcm.GetClusterStatus(), metav1.Now())

	// if True(v1alpha1.SyncChecked, tcm.GetClusterConditions()) {
	// 	return nil
	// }

	klog.Info("start master condition verify")
	if err := tcm.masterCondition.Verify(ctx); err != nil {
		klog.Errorf("master condition verify error: %v", err)
		return err
	}

	klog.Info("start executor condition verify")
	if err := tcm.executorCondition.Verify(ctx); err != nil {
		klog.Errorf("executor condition verify error: %v", err)
		return err
	}

	return tcm.Apply()
}

func (tcm *TiflowClusterConditionManager) Apply() error {
	tcm.GetClusterStatus().ServerName = tiflowapi.GetMasterClient(tcm.cli,
		tcm.GetNamespace(), tcm.GetName(), "", tcm.IsClusterTLSEnabled()).
		GetURL()

	SetTrue(v1alpha1.SyncChecked, tcm.GetClusterStatus(), metav1.Now())
	SetTrue(v1alpha1.VersionChecked, tcm.GetClusterStatus(), metav1.Now())
	return nil
}
