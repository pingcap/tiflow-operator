package condition

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/result"
	"github.com/pingcap/tiflow-operator/pkg/tiflowapi"
)

type ExecutorConditionManager struct {
	cli       client.Client
	clientSet kubernetes.Interface
	cluster   *v1alpha1.TiflowCluster
}

func NewExecutorConditionManager(cli client.Client, clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) ClusterCondition {
	return &ExecutorConditionManager{
		cli:       cli,
		clientSet: clientSet,
		cluster:   tc,
	}
}

func (ecm *ExecutorConditionManager) Update(ctx context.Context) error {
	SetFalse(v1alpha1.ExecutorSynced, &ecm.cluster.Status, metav1.Now())
	SetFalse(v1alpha1.ExecutorsInfoUpdatedChecked, &ecm.cluster.Status, metav1.Now())

	if err := ecm.update(ctx); err != nil {
		return result.SyncStatusErr{
			Err: err,
		}
	}

	return nil
}

func (ecm *ExecutorConditionManager) update(ctx context.Context) error {
	ns := ecm.cluster.GetNamespace()
	tcName := ecm.cluster.GetName()

	sts, err := ecm.clientSet.AppsV1().StatefulSets(ns).
		Get(ctx, executorMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		} else {
			return fmt.Errorf("executor [%s/%s] codition get satatefulSet error: %v",
				ns, tcName, err)
		}
	}
	ecm.cluster.Status.Executor.StatefulSet = &sts.Status

	if err = ecm.syncMembersStatus(ctx, sts); err != nil {
		return err
	}

	// get follows from podName
	ecm.cluster.Status.Executor.Image = ""
	if c := findContainerByName(sts, "tiflow-executor"); c != nil {
		ecm.cluster.Status.Executor.Image = c.Image
	}

	// todo: Need to get the info of volumes which running container has bound
	// todo: Waiting for discussion
	ecm.cluster.Status.Executor.Volumes = nil

	SetTrue(v1alpha1.ExecutorsInfoUpdatedChecked, &ecm.cluster.Status, metav1.Now())
	return nil
}

func (ecm *ExecutorConditionManager) Check() error {
	if ecm.cluster.Spec.Executor == nil {
		// todo: more gracefully
		SetTrue(v1alpha1.ExecutorNumChecked, &ecm.cluster.Status, metav1.Now())
		SetTrue(v1alpha1.ExecutorReadyChecked, &ecm.cluster.Status, metav1.Now())
		return nil
	}

	ns := ecm.cluster.GetNamespace()
	tcName := ecm.cluster.GetName()

	infosUpdateChecked := True(v1alpha1.ExecutorsInfoUpdatedChecked, ecm.cluster.Status.ClusterConditions)
	if !infosUpdateChecked {
		return result.SyncConditionErr{
			Err: fmt.Errorf("executor [%s/%s] check: information update failed", ns, tcName),
		}
	}

	if ecm.versionCheck() {
		SetTrue(v1alpha1.ExecutorVersionChecked, &ecm.cluster.Status, metav1.Now())
	} else {
		SetFalse(v1alpha1.ExecutorVersionChecked, &ecm.cluster.Status, metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("executor [%s/%s] check: version are not up-to-date", ns, tcName),
		}
	}

	// todo: need to check this
	if ecm.pvcCheck() {
		SetTrue(v1alpha1.ExecutorPVCChecked, &ecm.cluster.Status, metav1.Now())
	} else {
		SetFalse(v1alpha1.ExecutorPVCChecked, &ecm.cluster.Status, metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("executor [%s/%s] check: pvc's status is abnormal", ns, tcName),
		}
	}

	actual := ecm.cluster.ExecutorStsCurrentReplicas()
	desired := ecm.cluster.ExecutorStsDesiredReplicas()
	// todo: need to handle failureMembers
	// failed := len(ecm.cluster.Status.Executor.FailureMembers)

	if actual == desired {
		SetTrue(v1alpha1.ExecutorNumChecked, &ecm.cluster.Status, metav1.Now())
	} else {
		SetFalse(v1alpha1.ExecutorNumChecked, &ecm.cluster.Status, metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("executor [%s/%s] check: actual is not equal to desired replicas ", ns, tcName),
		}
	}

	// todo: need to check this
	if ecm.cluster.AllExecutorMembersReady() {
		SetTrue(v1alpha1.ExecutorReadyChecked, &ecm.cluster.Status, metav1.Now())
	} else {
		SetFalse(v1alpha1.ExecutorReadyChecked, &ecm.cluster.Status, metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("executor [%s/%s] check: cluster are not reday", ns, tcName),
		}
	}

	return nil
}

func (ecm *ExecutorConditionManager) syncMembersStatus(ctx context.Context, sts *appsv1.StatefulSet) error {
	ns := ecm.cluster.GetNamespace()
	tcName := ecm.cluster.GetName()

	if ecm.cluster.Heterogeneous() && ecm.cluster.WithoutLocalMaster() {
		ns = ecm.cluster.Spec.Cluster.Namespace
		tcName = ecm.cluster.Spec.Cluster.Name
	}

	tiflowClient := tiflowapi.GetMasterClient(ecm.cli, ns, tcName, "", ecm.cluster.IsClusterTLSEnabled())

	// get executors info from master
	executorsInfo, err := tiflowClient.GetExecutors()
	if err != nil {
		SetFalse(v1alpha1.ExecutorsInfoUpdatedChecked, &ecm.cluster.Status, metav1.Now())
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

func (ecm *ExecutorConditionManager) updateMembersInfo(executorsInfo tiflowapi.ExecutorsInfo) error {
	ns := ecm.cluster.GetNamespace()

	// todo: WIP, get information about the FailureMembers and FailoverUID through the MasterClient
	members := make(map[string]v1alpha1.ExecutorMember)
	for _, e := range executorsInfo.Executors {
		// todo: WIP
		if !strings.Contains(e.Address, ns) {
			continue
		}

		c, err := handleCapability(e.Capability)
		if err != nil {
			return err
		}

		memberName, err := formatName(e.Address)
		if err != nil {
			return err
		}

		members[memberName] = v1alpha1.ExecutorMember{
			Id:                 e.ID,
			Name:               e.Name,
			Addr:               e.Address,
			Capability:         c,
			LastTransitionTime: metav1.Now(),
		}
	}

	ecm.cluster.Status.Executor.Members = members
	return nil
}

func (ecm *ExecutorConditionManager) pvcCheck() bool {
	// todo: need to check pvc's status here
	return true
}

func (ecm *ExecutorConditionManager) versionCheck() bool {
	return statefulSetUpToDate(ecm.cluster.Status.Executor.StatefulSet, true)
}
