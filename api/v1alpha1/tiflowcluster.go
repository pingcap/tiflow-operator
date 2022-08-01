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

func (tc *TiflowCluster) WorkerImage() string {
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

func NewGenericConfig() *config.GenericConfig {
	return config.New(map[string]interface{}{})
}
