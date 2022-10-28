package testutil

import (
	"github.com/pingcap/tiflow-operator/api/config"
	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterBuilder struct {
	cluster v1alpha1.TiflowCluster
}

func NewBuilder(name string) ClusterBuilder {
	b := ClusterBuilder{
		cluster: v1alpha1.TiflowCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
			},
			Spec: v1alpha1.TiflowClusterSpec{
				Master:   &v1alpha1.MasterSpec{},
				Executor: &v1alpha1.ExecutorSpec{},
			},
		},
	}

	return b
}

func (b ClusterBuilder) WithNamespace(namespace string) ClusterBuilder {
	b.cluster.Namespace = namespace
	return b
}

func (b ClusterBuilder) WithMasterReplica(c int32) ClusterBuilder {
	b.cluster.Spec.Master.Replicas = c
	return b
}

func (b ClusterBuilder) WithExecutorReplica(c int32) ClusterBuilder {
	b.cluster.Spec.Executor.Replicas = c
	return b
}

func (b ClusterBuilder) WithMasterImage(image string) ClusterBuilder {
	b.cluster.Spec.Master.BaseImage = image
	return b
}

func (b ClusterBuilder) WithExecutorImage(image string) ClusterBuilder {
	b.cluster.Spec.Executor.BaseImage = image
	return b
}

func (b ClusterBuilder) WithVersion(version string) ClusterBuilder {
	b.cluster.Spec.Version = version
	return b
}

func (b ClusterBuilder) WithMasterConfig(cfg *config.GenericConfig) ClusterBuilder {
	b.cluster.Spec.Master.Config = cfg
	return b
}

func (b ClusterBuilder) Cr() *v1alpha1.TiflowCluster {
	cluster := b.cluster.DeepCopy()

	return cluster
}
