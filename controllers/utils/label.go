package utils

const (
	// The following labels are recommended by kubernetes https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/

	// ManagedByLabelKey is Kubernetes recommended label key, it represents the tool being used to manage the operation of an application
	// For resources managed by TiFlow Operator, its value is always tiflow-operator
	ManagedByLabelKey string = "app.kubernetes.io/managed-by"
	// ComponentLabelKey is Kubernetes recommended label key, it represents the component within the architecture
	ComponentLabelKey string = "app.kubernetes.io/component"
	// NameLabelKey is Kubernetes recommended label key, it represents the name of the application
	NameLabelKey string = "app.kubernetes.io/name"
	// InstanceLabelKey is Kubernetes recommended label key, it represents a unique name identifying the instance of an application
	// It's set by helm when installing a release
	InstanceLabelKey string = "app.kubernetes.io/instance"

	// NamespaceLabelKey is label key used in PV for easy querying
	NamespaceLabelKey string = "app.kubernetes.io/namespace"
	// UsedByLabelKey indicate where it is used. for example, tidb has two services,
	// one for internal component access and the other for end-user
	UsedByLabelKey string = "app.kubernetes.io/used-by"
	// ClusterIDLabelKey is cluster id label key
	ClusterIDLabelKey string = "tidb.pingcap.com/cluster-id"
	// StoreIDLabelKey is store id label key
	StoreIDLabelKey string = "tidb.pingcap.com/store-id"
	// MemberIDLabelKey is member id label key
	MemberIDLabelKey string = "tidb.pingcap.com/member-id"

	// TiFlowOperator is ManagedByLabelKey label value
	TiFlowOperator string = "tiflow-operator"

	// ExecutorLabelVal is tiflow-executor label value
	ExecutorLabelVal string = "tiflow-executor"
)

type Label map[string]string

func (l Label) Instance(name string) Label {
	l[InstanceLabelKey] = name
	return l
}

func NewTiflowCluster() Label {
	return Label{
		NameLabelKey:      "tiflow-cluster",
		ManagedByLabelKey: TiFlowOperator,
	}
}

func (l Label) Component(name string) Label {
	l[ComponentLabelKey] = name
	return l
}

func (l Label) Executor() Label {
	return l.Component(ExecutorLabelVal)
}

func (l Label) Labels() map[string]string {
	return l
}
