package v1alpha1

import (
	"fmt"
	"github.com/StepOnce7/tiflow-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
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
	//if tc.IsTLSClusterEnabled() {
	//	return "https"
	//}
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

func (tc *TiflowCluster) ExecutorAllMembersReady() bool {
	if int(tc.ExecutorStsDesiredReplicas()) != len(tc.Status.Executor.Members) {
		return false
	}

	//for _, member := range tc.Status.Executor.Members {
	//	if member {
	//
	//	}
	//}
	return true
}

func (tc *TiflowCluster) ExecutorStsActualReplicas() int32 {
	stsStatus := tc.Status.Executor.StatefulSet
	if stsStatus == nil {
		return 0
	}

	return stsStatus.Replicas
}

func (tc *TiflowCluster) ExecutorStsDesiredReplicas() int32 {
	if tc.Spec.Executor == nil {
		return 0
	}

	return tc.Spec.Executor.Replicas + int32(len(tc.Status.Executor.FailureMembers))
}

func (mt MemberType) String() string {
	return string(mt)
}

func NewGenericConfig() *config.GenericConfig {
	return config.New(map[string]interface{}{})
}
