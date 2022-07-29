package v1alpha1

import (
	"fmt"
)

func (tc *TiflowCluster) GetInstanceName() string {
	return tc.Name
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

func (tc *TiflowCluster) BaseExecutorSpec() ComponentAccessor {
	var spec *ComponentSpec
	if tc.Spec.Executor != nil {
		spec = &tc.Spec.Executor.ComponentSpec
	}

	return buildTiFLowClusterComponentAccessor(TiFlowExecutorMemberType, tc, spec)
}
