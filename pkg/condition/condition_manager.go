package condition

import (
	"context"
)

type Condition interface {
	CheckCondition
	Sync(context.Context) error
	Apply(context.Context) error
}

type ClusterCondition interface {
	UpdateCondition
	CheckCondition
}

type UpdateCondition interface {
	Update(context.Context) error
}

type CheckCondition interface {
	Check(context.Context) error
}
