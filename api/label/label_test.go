package label

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/labels"
)

func TestLabelNew(t *testing.T) {
	l := New()
	require.Equal(t, "tiflow-cluster", l[NameLabelKey])
	require.Equal(t, "tiflow-operator", l[ManagedByLabelKey])
}

func TestLabelInstance(t *testing.T) {
	l := New()
	l.Instance("demo")
	require.Equal(t, "demo", l[InstanceLabelKey])
}

func TestLabelNamespace(t *testing.T) {
	l := New()
	l.Namespace("ns-1")
	require.Equal(t, "ns-1", l[NamespaceLabelKey])
}

func TestLabelComponent(t *testing.T) {
	l := New()
	l.Component("tiflow-executor")
	require.Equal(t, "tiflow-executor", l[ComponentLabelKey])
}

func TestLabelTiflowMaster(t *testing.T) {
	l := New()
	l.TiflowMaster()
	require.True(t, l.IsTiflowMaster())
}

func TestLabelDMWorker(t *testing.T) {
	l := New()
	l.TiflowExecutor()
	require.True(t, l.IsTiflowExecutor())
}

func TestLabelSelector(t *testing.T) {
	l := New()
	l.TiflowMaster()
	l.Instance("demo")
	l.Namespace("ns-1")
	s, err := l.Selector()
	require.NoError(t, err)
	m := map[string]string{
		NameLabelKey:      "tiflow-cluster",
		ManagedByLabelKey: "tiflow-operator",
		ComponentLabelKey: "tiflow-master",
		InstanceLabelKey:  "demo",
		NamespaceLabelKey: "ns-1",
	}
	st := labels.Set(m)
	require.True(t, s.Matches(st))

	ls := l.LabelSelector()
	require.Equal(t, m, ls.MatchLabels)
}

func TestLabelLabels(t *testing.T) {
	l := New()
	l.TiflowExecutor()
	l.Instance("demo")
	l.Namespace("ns-1")
	ls := l.Labels()
	m := map[string]string{
		NameLabelKey:      "tiflow-cluster",
		ManagedByLabelKey: "tiflow-operator",
		ComponentLabelKey: "tiflow-executor",
		InstanceLabelKey:  "demo",
		NamespaceLabelKey: "ns-1",
	}
	require.Equal(t, m, ls)
}
