package v1alpha1

import (
	"fmt"

	"github.com/pingcap/tiflow-operator/api/config"
	"github.com/pingcap/tiflow-operator/api/label"
)

func (tc *TiflowCluster) GetInstanceName() string {
	labels := tc.GetLabels()
	if inst, ok := labels[label.InstanceLabelKey]; ok {
		return inst
	}
	return tc.Name
}

func (tc *TiflowCluster) Scheme() string {
	// TODO: tls
	// if tc.IsTLSClusterEnabled() {
	//	return "https"
	// }
	return "http"
}

func (tc *TiflowCluster) MasterImage() string {
	image := tc.Spec.Master.BaseImage
	version := tc.Spec.Master.Version
	if version == nil {
		version = &tc.Spec.Version
	}
	if *version != "" {
		image = fmt.Sprintf("%s:%s", image, *version)
	}
	return image
}
func (tc *TiflowCluster) ExecutorImage() string {
	image := tc.Spec.Executor.BaseImage
	version := tc.Spec.Executor.Version
	if version == nil {
		version = &tc.Spec.Version
	}

	if *version != "" {
		image = fmt.Sprintf("%s:%s", image, *version)
	}
	return image
}

func (tc *TiflowCluster) AllExecutorMembersReady() bool {
	return int(tc.ExecutorStsDesiredReplicas()) == len(tc.Status.Executor.Members)
}

func (tc *TiflowCluster) ExecutorUpgrading() bool {
	return tc.Status.Executor.Phase == ExecutorUpgrading
}

func (tc *TiflowCluster) ExecutorScaling() bool {
	return tc.Status.Executor.Phase == ExecutorScalingIn || tc.Status.Executor.Phase == ExecutorScalingOut
}

func (tc *TiflowCluster) ExecutorStsActualReplicas() int32 {
	stsStatus := tc.Status.Executor.StatefulSet
	if stsStatus == nil {
		return 0
	}

	return stsStatus.Replicas
}

func (tc *TiflowCluster) ExecutorStsReadyReplicas() int32 {
	stsStatus := tc.Status.Executor.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.ReadyReplicas
}

func (tc *TiflowCluster) ExecutorStsCurrentReplicas() int32 {
	stsStatus := tc.Status.Executor.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.CurrentReplicas
}

func (tc *TiflowCluster) ExecutorStsUpdatedReplicas() int32 {
	stsStatus := tc.Status.Executor.StatefulSet
	if stsStatus == nil {
		return 0
	}

	return stsStatus.UpdatedReplicas
}

func (tc *TiflowCluster) ExecutorStsDesiredReplicas() int32 {
	if tc.Spec.Executor == nil {
		return 0
	}

	return tc.Spec.Executor.Replicas + int32(len(tc.Status.Executor.FailureMembers))
}
func (tc *TiflowCluster) ExecutorAllMembers() int32 {
	if tc.Spec.Executor != nil {
		return tc.ExecutorActualMembers() + tc.ExecutorActualPeerMembers()
	}
	return 0
}

func (tc *TiflowCluster) ExecutorActualMembers() int32 {
	if tc.Status.Executor.Members != nil {
		return int32(len(tc.Status.Executor.Members))
	}
	return 0
}

func (tc *TiflowCluster) ExecutorActualPeerMembers() int32 {
	if tc.Status.Executor.PeerMembers != nil {
		return int32(len(tc.Status.Executor.PeerMembers))
	}
	return 0
}

func (tc *TiflowCluster) MasterUpgrading() bool {
	return tc.Status.Master.Phase == MasterUpgrading
}

func (tc *TiflowCluster) MasterScaling() bool {
	return tc.Status.Master.Phase == MasterScalingIn || tc.Status.Master.Phase == MasterScalingOut
}

func (tc *TiflowCluster) MasterStsActualReplicas() int32 {
	stsStatus := tc.Status.Master.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.Replicas
}

func (tc *TiflowCluster) MasterStsReadyReplicas() int32 {
	stsStatus := tc.Status.Master.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.ReadyReplicas
}

func (tc *TiflowCluster) MasterStsCurrentReplicas() int32 {
	stsStatus := tc.Status.Master.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.CurrentReplicas
}

func (tc *TiflowCluster) MasterStsUpdatedReplicas() int32 {
	stsStatus := tc.Status.Master.StatefulSet
	if stsStatus == nil {
		return 0
	}

	return stsStatus.UpdatedReplicas
}

func (tc *TiflowCluster) MasterStsDesiredReplicas() int32 {
	return tc.Spec.Master.Replicas + int32(len(tc.Status.Master.FailureMembers))
}

func (tc *TiflowCluster) MasterAllMembers() int32 {
	if tc.Spec.Master != nil {
		return tc.MasterActualMembers() + tc.MasterActualPeerMembers()
	}
	return 0
}

func (tc *TiflowCluster) MasterActualMembers() int32 {
	if tc.Status.Master.Members != nil {
		return int32(len(tc.Status.Master.Members))
	}
	return 0
}

func (tc *TiflowCluster) MasterActualPeerMembers() int32 {
	if tc.Status.Master.PeerMembers != nil {
		return int32(len(tc.Status.Master.PeerMembers))
	}
	return 0
}

func (tc *TiflowCluster) IsClusterTLSEnabled() bool {
	return tc.Spec.TLSCluster != nil && *tc.Spec.TLSCluster
}

func (tc *TiflowCluster) AllMasterMembersReady() bool {
	return int(tc.MasterStsDesiredReplicas()) == len(tc.Status.Master.Members)
}

func (tc *TiflowCluster) Heterogeneous() bool {
	return tc.Spec.Cluster != nil && len(tc.Spec.Cluster.Name) > 0
}

func (tc *TiflowCluster) WithoutLocalMaster() bool {
	return tc.Spec.Master == nil
}

func (tc *TiflowCluster) MasterIsAvailable() bool {
	return tc.Status.Master.Leader.Id != ""
}

func (mt MemberType) String() string {
	return string(mt)
}

func NewGenericConfig() *config.GenericConfig {
	return config.New(map[string]interface{}{})
}

func (tc *TiflowCluster) GetClusterStatus() *TiflowClusterStatus {
	return &tc.Status
}

func (tc *TiflowCluster) GetClusterConditions() []TiflowClusterCondition {
	return tc.Status.ClusterConditions
}

func (tc *TiflowCluster) GetMasterStatus() *MasterStatus {
	return &tc.Status.Master
}

func (tc *TiflowCluster) GetExecutorStatus() *ExecutorStatus {
	return &tc.Status.Executor
}

func (tc *TiflowCluster) GetMasterSyncTypes() []ClusterSyncType {
	return tc.Status.Master.SyncTypes
}

func (tc *TiflowCluster) GetExecutorSyncTypes() []ClusterSyncType {
	return tc.Status.Executor.SyncTypes
}

func (tc *TiflowCluster) GetMasterPhase() MasterPhaseType {
	return tc.Status.Master.Phase
}

func (tc *TiflowCluster) GetExecutorPhase() ExecutorPhaseType {
	return tc.Status.Executor.Phase
}

func (tc *TiflowCluster) GetClusterPhase() TiflowClusterPhaseType {
	return tc.Status.ClusterPhase
}
