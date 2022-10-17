package condition

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
)

type Condition interface {
	Sync(context.Context) error
	Apply() error
}

type ClusterCondition interface {
	UpdateCondition
	VerifyCondition
}

type UpdateCondition interface {
	Update(context.Context, *appsv1.StatefulSet) error
}

type VerifyCondition interface {
	Verify(context.Context) error
}
