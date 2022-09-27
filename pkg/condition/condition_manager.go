package condition

import (
	"context"
)

type Condition interface {
	Sync(context.Context) error
	Apply() error
}

type ClusterCondition interface {
	UpdateCondition
	CheckCondition
}

type UpdateCondition interface {
	Update(context.Context) error
}

type CheckCondition interface {
	Check() error
}
