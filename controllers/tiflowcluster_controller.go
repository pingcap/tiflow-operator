/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/pingcap/tiflow-operator/pkg/controller"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pingcapcomv1alpha1 "github.com/pingcap/tiflow-operator/api/v1alpha1"
)

// TiflowClusterReconciler reconciles a TiflowCluster object
type TiflowClusterReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Control controller.ControlInterface
}

func NewTiflowClusterReconciler(cli client.Client, clientSet kubernetes.Interface, scheme *runtime.Scheme) *TiflowClusterReconciler {
	return &TiflowClusterReconciler{
		Client:  cli,
		Scheme:  scheme,
		Control: controller.NewDefaultTiflowClusterControl(cli, clientSet),
	}
}

//+kubebuilder:rbac:groups=pingcap.com,resources=tiflowclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pingcap.com,resources=tiflowclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pingcap.com,resources=tiflowclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps;secrets;services;persistentvolumeclaims;persistentvolumes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;delete
//+kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=statefulsets;deployments,verbs=*
//+kubebuilder:rbac:groups=apps,resources=statefulsets/scale,verbs=get;watch;update
//+kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;update;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;update;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=*
//+kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TiflowCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *TiflowClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	tc := &pingcapcomv1alpha1.TiflowCluster{}

	if err := r.Get(ctx, req.NamespacedName, tc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	if err := r.Control.UpdateTiflowCluster(ctx, tc); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TiflowClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pingcapcomv1alpha1.TiflowCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
