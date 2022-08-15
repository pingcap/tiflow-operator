package member

import (
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type masterScaler struct {
	cli client.Client
}

// NewMasterScaler returns a DMScaler
func NewMasterScaler(cli client.Client) Scaler {
	return &masterScaler{
		cli: cli,
	}
}

func (s masterScaler) Scale(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(actual, desired)
	if scaling > 0 {
		return s.ScaleOut(meta, actual, desired)
	} else if scaling < 0 {
		return s.ScaleIn(meta, actual, desired)
	}
	return nil
}

func (s masterScaler) ScaleOut(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error {
	//TODO implement me
	panic("implement me")
}

func (s masterScaler) ScaleIn(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error {
	//TODO implement me
	panic("implement me")
}
