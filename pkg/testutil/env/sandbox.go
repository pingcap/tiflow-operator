package env

import (
	"context"
	"fmt"
	"testing"

	"github.com/pingcap/errors"
	"github.com/pingcap/tiflow-operator/controllers"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	DefaultNsName = "tiflow-test-"
)

func NewSandbox(t *testing.T, env *ActiveEnv) Sandbox {
	ns := DefaultNsName + rand.String(6)

	mgr, err := ctrl.NewManager(env.k8s.Cfg, ctrl.Options{
		Scheme:             env.scheme,
		Namespace:          ns,
		MetricsBindAddress: "0", // disable metrics serving
	})
	require.NoError(t, err)

	s := Sandbox{
		env:       env,
		Namespace: ns,
		Mgr:       mgr,
	}

	require.NoError(t, createNamespace(s))
	t.Cleanup(s.Cleanup)

	return s
}

type Sandbox struct {
	env *ActiveEnv
	Mgr ctrl.Manager

	Namespace string
}

func (s Sandbox) setNamespaceIfMissing(obj client.Object) client.Object {
	accessor, _ := meta.Accessor(obj)
	if accessor.GetNamespace() == "" {
		runtimeObject := obj.DeepCopyObject()
		accessor, _ = meta.Accessor(runtimeObject)
		accessor.SetNamespace(s.Namespace)
		obj, _ := runtimeObject.(client.Object)
		return obj
	}

	return obj
}

func (s Sandbox) Create(obj client.Object) error {
	obj = s.setNamespaceIfMissing(obj)

	return s.env.Create(context.TODO(), obj)
}

func (s Sandbox) Update(obj client.Object) error {
	obj = s.setNamespaceIfMissing(obj)

	return s.env.Update(context.TODO(), obj)
}

func (s Sandbox) Patch(obj client.Object, patch client.Patch) error {
	obj = s.setNamespaceIfMissing(obj)

	return s.env.Patch(context.TODO(), obj, patch)
}

func (s Sandbox) Delete(obj client.Object) error {
	obj = s.setNamespaceIfMissing(obj)

	return s.env.Delete(context.TODO(), obj)
}

func (s Sandbox) Get(o client.Object) error {
	accessor, err := meta.Accessor(o)
	if err != nil {
		return err
	}

	key := types.NamespacedName{
		Namespace: s.Namespace,
		Name:      accessor.GetName(),
	}

	return s.env.Get(context.TODO(), key, o)
}

func (s Sandbox) List(list client.ObjectList, labels map[string]string) error {
	ns := client.InNamespace(s.Namespace)
	matchingLabels := client.MatchingLabels(labels)

	return s.env.List(context.TODO(), list, ns, matchingLabels)
}

func (s Sandbox) Cleanup() {
	dp := metav1.DeletePropagationForeground
	opts := metav1.DeleteOptions{PropagationPolicy: &dp}
	nss := s.env.Clientset.CoreV1().Namespaces()
	if err := nss.Delete(context.TODO(), s.Namespace, opts); err != nil {
		fmt.Println("failed to cleanup namespace", s.Namespace)
	}
}

func (s Sandbox) StartManager(t *testing.T) {
	clientSet, err := kubernetes.NewForConfig(s.Mgr.GetConfig())
	require.NoError(t, err)

	reconciler := controllers.NewTiflowClusterReconciler(s.Mgr.GetClient(), clientSet, s.Mgr.GetScheme())
	require.NoError(t, reconciler.SetupWithManager(s.Mgr))

	standaloneReconcile := controllers.NewStandaloneReconciler(s.Mgr.GetClient(), s.Mgr.GetScheme())
	require.NoError(t, standaloneReconcile.SetupWithManager(s.Mgr), err)

	t.Cleanup(startCtrlMgr(t, s.Mgr))
}

func createNamespace(s Sandbox) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.Namespace,
		},
	}

	if _, err := s.env.Clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{}); err != nil {
		return errors.Wrapf(err, "failed to create namespace: %s", s.Namespace)
	}

	return nil
}

// nolint
func startCtrlMgr(t *testing.T, mgr manager.Manager) func() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		require.NoError(t, mgr.Start(ctx))
	}()

	return cancel
}
