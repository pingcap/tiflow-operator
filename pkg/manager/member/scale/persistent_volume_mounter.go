package scale

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PersistentVolumeMounter provides Mounter() method to add existing but unused statefulset
// PVC and their underlying PVs.
// Only use, if the executor is stateful
type PersistentVolumeMounter struct {
	Client      client.Client
	Namespace   string
	StatefulSet string
}

func NewPersistentVolumeMounter(cli client.Client, namespace string, stsName string) PVCMounter {
	return &PersistentVolumeMounter{
		cli,
		namespace,
		stsName,
	}
}

func (p *PersistentVolumeMounter) Mounter(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (p *PersistentVolumeMounter) IsStateful() bool {
	//TODO implement me
	panic("implement me")
}
