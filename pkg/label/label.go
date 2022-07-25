package label

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// The following labels are recommended by kubernetes https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/

	// ManagedByLabelKey is Kubernetes recommended label key, it represents the tool being used to manage the operation of an application
	// For resources managed by TiDB Operator, its value is always tidb-operator
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

	// TiFlowOperator is ManagedByLabelKey label value
	TiFlowOperator string = "tiflow-operator"

	// AnnForceUpgradeKey is tc annotation key to indicate whether force upgrade should be done
	AnnForceUpgradeKey = "tidb.pingcap.com/force-upgrade"
	// AnnForceUpgradeVal is tc annotation value to indicate whether force upgrade should be done
	AnnForceUpgradeVal = "true"

	// TiflowMasterLabelVal is tiflow-master label value
	TiflowMasterLabelVal string = "tiflow-master"
	// TiflowExecutorLabelVal is tiflow-executor label value
	TiflowExecutorLabelVal string = "tiflow-executor"

	// AnnTiflowMasterDeleteSlots is annotation key of tiflow-master delete slots.
	AnnTiflowMasterDeleteSlots = "tiflow-master.tidb.pingcap.com/delete-slots"
	// AnnTiflowExecutorDeleteSlots is annotation key of tiflow-executor delete slots.
	AnnTiflowExecutorDeleteSlots = "tiflow-executor.tidb.pingcap.com/delete-slots"
)

// Label is the label field in metadata
type Label map[string]string

// New initialize a new Label for components of tidb cluster
func New() Label {
	return Label{
		NameLabelKey:      "tiflow-cluster",
		ManagedByLabelKey: TiFlowOperator,
	}
}

// Instance adds instance kv pair to label
func (l Label) Instance(name string) Label {
	l[InstanceLabelKey] = name
	return l
}

// UsedBy adds use-by kv pair to label
func (l Label) UsedBy(name string) Label {
	l[UsedByLabelKey] = name
	return l
}

// UsedByPeer adds used-by=peer label
func (l Label) UsedByPeer() Label {
	l[UsedByLabelKey] = "peer"
	return l
}

// UsedByEndUser adds use-by=end-user label
func (l Label) UsedByEndUser() Label {
	l[UsedByLabelKey] = "end-user"
	return l
}

// Namespace adds namespace kv pair to label
func (l Label) Namespace(name string) Label {
	l[NamespaceLabelKey] = name
	return l
}

// Component adds component kv pair to label
func (l Label) Component(name string) Label {
	l[ComponentLabelKey] = name
	return l
}

// TiflowMaster assigns tiflow-master to component key in label
func (l Label) TiflowMaster() Label {
	return l.Component(TiflowMasterLabelVal)
}

// IsTiflowMaster returns whether label is a TiflowMaster component
func (l Label) IsTiflowMaster() bool {
	return l[ComponentLabelKey] == TiflowMasterLabelVal
}

// TiflowExecutor assigns tiflow-executor to component key in label
func (l Label) TiflowExecutor() Label {
	return l.Component(TiflowExecutorLabelVal)
}

// IsTiflowExecutor returns whether label is a TiflowExecutor component
func (l Label) IsTiflowExecutor() bool {
	return l[ComponentLabelKey] == TiflowExecutorLabelVal
}

// Selector gets labels.Selector from label
func (l Label) Selector() (labels.Selector, error) {
	return metav1.LabelSelectorAsSelector(l.LabelSelector())
}

// LabelSelector gets LabelSelector from label
func (l Label) LabelSelector() *metav1.LabelSelector {
	return &metav1.LabelSelector{MatchLabels: l}
}

// Labels converts label to map[string]string
func (l Label) Labels() map[string]string {
	return l
}

// Copy copy the value of label to avoid pointer copy
func (l Label) Copy() Label {
	copyLabel := make(Label)
	for k, v := range l {
		copyLabel[k] = v
	}
	return copyLabel
}
