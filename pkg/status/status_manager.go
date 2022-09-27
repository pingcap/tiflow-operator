package status

import "github.com/pingcap/tiflow-operator/api/v1alpha1"

type SyncTypeManager interface {
	Ongoing(v1alpha1.SyncTypeName, string)
	Complied(v1alpha1.SyncTypeName, string)
	Failed(v1alpha1.SyncTypeName, string)
	Unknown(v1alpha1.SyncTypeName, string)
}
