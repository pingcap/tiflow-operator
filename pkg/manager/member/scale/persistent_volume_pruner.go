package scale

import (
	"context"
	"github.com/pingcap/tiflow-operator/pkg/manager/member"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PersistentVolumePruner provides a Prune() method to remove unused statefulset
// PVC and their underlying PVs.
// The underlying PVs SHOULD have their reclaim policy set to delete.
type PersistentVolumePruner struct {
	Client      client.Client
	Namespace   string
	StatefulSet string
}

func NewPersistentVolumePruner(cli client.Client, namespace string, stsName string) member.PVCPruner {
	return &PersistentVolumePruner{
		cli,
		namespace,
		stsName,
	}
}

func (p *PersistentVolumePruner) Prune(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}
