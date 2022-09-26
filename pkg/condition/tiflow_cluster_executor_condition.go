package condition

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/controller"
	mngerutils "github.com/pingcap/tiflow-operator/pkg/manager/utils"
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
	SetFalse(v1alpha1.MasterSynced, &ecm.cluster.Status, metav1.Now())
	ecm.cluster.Status.Executor.Synced = false

	ns := ecm.cluster.GetNamespace()
	tcName := ecm.cluster.GetName()

	sts, err := ecm.clientSet.AppsV1().StatefulSets(ns).
		Get(ctx, controller.TiflowExecutorMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		} else {
			return fmt.Errorf("executor [%s/%s] codition get satatefulSet error: %v",
				ns, tcName, err)
		}
	}

	upgrading, err := ecm.statsfulSetIsUpgrading(ctx, sts)
	if err != nil {
		return err
	}
	// todo: remove this logic to status pkg
	if ecm.cluster.ExecutorStsDesiredReplicas() != *sts.Spec.Replicas {
		if ecm.cluster.ExecutorStsDesiredReplicas() > *sts.Spec.Replicas {
			ecm.cluster.Status.Executor.Phase = v1alpha1.ExecutorScalingOut
		}
		ecm.cluster.Status.Executor.Phase = v1alpha1.ExecutorScalingIn
	} else if upgrading {
		ecm.cluster.Status.Executor.Phase = v1alpha1.ExecutorUpgrading
	} else {
		ecm.cluster.Status.Executor.Phase = v1alpha1.ExecutorRunning
	}

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

func (ecm *ExecutorConditionManager) Check(ctx context.Context) error {
	if ecm.cluster.Spec.Executor == nil {
		// todo: more gracefully
		SetTrue(v1alpha1.ExecutorNumChecked, &ecm.cluster.Status, metav1.Now())
		SetTrue(v1alpha1.ExecutorReadyChecked, &ecm.cluster.Status, metav1.Now())
		SetTrue(v1alpha1.ExecutorSynced, &ecm.cluster.Status, metav1.Now())
		return nil
	}

	ns := ecm.cluster.GetNamespace()
	tcName := ecm.cluster.GetName()

	infosUpdateChecked := True(v1alpha1.ExecutorsInfoUpdatedChecked, ecm.cluster.Status.ClusterConditions)
	if !infosUpdateChecked {
		return result.UpdateClusterStatus{
			Err: fmt.Errorf("executor [%s/%s] check: information update failed", ns, tcName),
		}
	}

	actual := int32(len(ecm.cluster.Status.Master.Members))
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

	if ecm.cluster.AllMasterMembersReady() {
		SetTrue(v1alpha1.ExecutorReadyChecked, &ecm.cluster.Status, metav1.Now())
	} else {
		SetFalse(v1alpha1.ExecutorReadyChecked, &ecm.cluster.Status, metav1.Now())
		return result.NotReadyErr{
			Err: fmt.Errorf("executor [%s/%s] check: executor not reday", ns, tcName),
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
				ns, tcName, controller.TiflowExecutorMemberName(tcName), err, epErr)
		}

		// tiflow-executor service has no endpoints
		if eps != nil && eps.Items != nil && len(eps.Items[0].Subsets) == 0 {
			return fmt.Errorf("%s, service %s/%s has no endpoints", err, ns, controller.TiflowExecutorMemberName(tcName))
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

		memberName, err := formatExecutorName(e.Address)
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

func (ecm *ExecutorConditionManager) statsfulSetIsUpgrading(ctx context.Context, sts *appsv1.StatefulSet) (bool, error) {
	if mngerutils.StatefulSetIsUpgrading(sts) {
		return true, nil
	}

	ns := ecm.cluster.GetNamespace()
	instanceName := ecm.cluster.GetInstanceName()
	selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	if err != nil {
		return false, fmt.Errorf("executor [%s/%s] condition converting selector error: %v",
			ns, instanceName, err)
	}
	executorPods, err := ecm.clientSet.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return false, fmt.Errorf("executor [%s/%s] condition listing master's pods error: %v",
			ns, instanceName, err)
	}

	for _, pod := range executorPods.Items {
		revisionHash, exist := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != ecm.cluster.Status.Master.StatefulSet.UpdateRevision {
			return true, nil
		}
	}
	return false, nil
}

func handleCapability(o string) (int64, error) {
	var i interface{}
	d := json.NewDecoder(strings.NewReader(o))
	d.UseNumber()

	if err := d.Decode(&i); err != nil {
		return -1, err
	}

	n := i.(json.Number)
	res, err := n.Int64()
	if err != nil {
		return -1, err
	}

	return res, nil
}

func formatExecutorName(name string) (string, error) {
	nameSlice := strings.Split(name, ".")
	if len(nameSlice) != 4 {
		return "", fmt.Errorf("split name %s error", name)
	}

	res := fmt.Sprintf("%s.%s", nameSlice[0], nameSlice[2])
	return res, nil
}
