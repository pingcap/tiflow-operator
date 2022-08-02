package controllers

import (
	"context"
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	FRAME_STANDALONE = "mysql"
	USER_STANDALONE  = "etcd"
	CONTAINER_PORT   = 3306
)

func (r *TiflowClusterReconciler) ReconcileFrameStandalone(ctx context.Context, instance *v1alpha1.TiflowCluster, req ctrl.Request) (ctrl.Result, error) {

	logg := log.FromContext(ctx)

	var pv corev1.PersistentVolume
	pv.Name = "mysql-pv-volume"
	createPVIfNotExists(&pv)
	err := r.Create(ctx, &pv)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			err = nil
		} else {
			logg.Error(err, "create mysql pv error")
			return ctrl.Result{}, err
		}
	}

	logg.Info("Create", "Mysql PV", "success")

	var cfm corev1.ConfigMap
	cfm.Name = "mysqlcfm"
	cfm.Namespace = instance.Namespace

	or, err := ctrl.CreateOrUpdate(ctx, r.Client, &cfm, func() error {
		createConfigMapIfNotExists(&cfm)
		return controllerutil.SetControllerReference(instance, &cfm, r.Scheme)
	})

	if err != nil {
		logg.Error(err, "create mysql configMap error")
		return ctrl.Result{}, err
	}
	logg.Info("CreateOrUpdate", "Mysql ConfigMap", or)

	var pvc corev1.PersistentVolumeClaim
	pvc.Name = "mysql-pv-claim"
	pvc.Namespace = instance.Namespace
	createPVCIfNotExists(&pvc)
	controllerutil.SetControllerReference(instance, &pvc, r.Scheme)
	err = r.Create(ctx, &pvc)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			err = nil
		} else {
			logg.Error(err, "create mysql pv error")
			return ctrl.Result{}, err
		}
	}

	logg.Info("Create", "Mysql PVC", "success")

	var svc corev1.Service
	svc.Name = FRAME_STANDALONE
	svc.Namespace = instance.Namespace

	or, err = ctrl.CreateOrUpdate(ctx, r.Client, &svc, func() error {
		createServiceIfNotExists(instance, &svc)
		return controllerutil.SetControllerReference(instance, &svc, r.Scheme)
	})

	if err != nil {
		logg.Error(err, "create mysql service error")
		return ctrl.Result{}, err
	}

	logg.Info("CreateOrUpdate", "Mysql Service", or)

	var deploy appsv1.Deployment
	deploy.Name = FRAME_STANDALONE
	deploy.Namespace = instance.Namespace
	or, err = ctrl.CreateOrUpdate(ctx, r.Client, &deploy, func() error {
		createDeploymentIfNotExists(instance, &deploy)
		return controllerutil.SetControllerReference(instance, &deploy, r.Scheme)
	})

	if err != nil {
		logg.Error(err, "create mysql deployment error")
		return ctrl.Result{}, err
	}

	logg.Info("CreateOrUpdate", "Mysql Deployment", or)

	logg.Info("frame standalone reconcile end", "reconcile", "success")

	return ctrl.Result{}, nil
}

func createConfigMapIfNotExists(cfm *corev1.ConfigMap) {
	cfm.Labels = map[string]string{
		FRAME_STANDALONE: "mysql",
	}

	cfm.Data = map[string]string{
		"my.cnf": `# Apply this config only on the mysql deployment.
[mysqld]
# Accept connections from any IP address
bind-address	            = 0.0.0.0
`,
	}
}

func createPVCIfNotExists(pvc *corev1.PersistentVolumeClaim) {

	pvc.Labels = map[string]string{
		FRAME_STANDALONE: "mysql",
	}

	scn := "mysql"
	pvc.Spec = corev1.PersistentVolumeClaimSpec{
		StorageClassName: &scn,
		AccessModes: []corev1.PersistentVolumeAccessMode{
			corev1.ReadWriteOnce,
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("5Gi"),
			},
		},
	}

}

func createServiceIfNotExists(de *v1alpha1.TiflowCluster, svc *corev1.Service) {

	svc.Labels = map[string]string{
		FRAME_STANDALONE: "mysql-standalone",
	}

	svc.Spec = corev1.ServiceSpec{
		ClusterIP: corev1.ClusterIPNone,
		Selector: map[string]string{
			"app": FRAME_STANDALONE,
		},
		Ports: []corev1.ServicePort{
			{
				Port: CONTAINER_PORT,
			},
		},
	}
}

func createDeploymentIfNotExists(de *v1alpha1.TiflowCluster, deploy *appsv1.Deployment) {

	deploy.Labels = map[string]string{
		FRAME_STANDALONE: "mysql-standalone",
	}

	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: pointer.Int32Ptr(1),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": FRAME_STANDALONE,
			},
		},
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": FRAME_STANDALONE,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "frame-mysql-standalone",
						Image:           "mysql:latest",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								Name:  "MYSQL_ROOT_PASSWORD",
								Value: "123456",
							},
						},
						Ports: []corev1.ContainerPort{
							{
								Name:          "mysql",
								ContainerPort: CONTAINER_PORT,
								Protocol:      corev1.ProtocolTCP,
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "mysql-persistent-storage",
								MountPath: "/var/lib/mysql",
							},
							{
								Name:      "config-map",
								MountPath: "/etc/my.cnf",
								SubPath:   "my.cnf",
							},
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"mysqladmin", "ping",
										"-h127.0.0.1", "-P3306",
										"-uroot", "-p123456",
									},
								},
							},
							InitialDelaySeconds: 30,
							PeriodSeconds:       10,
							TimeoutSeconds:      600,
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "mysql-persistent-storage",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "mysql-pv-claim",
							},
						},
					},
					{
						Name: "config-map",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "mysqlcfm",
								},
								Items: []corev1.KeyToPath{
									{
										Key:  "my.cnf",
										Path: "my.cnf",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func createPVIfNotExists(pv *corev1.PersistentVolume) {

	pv.Spec = corev1.PersistentVolumeSpec{
		StorageClassName: "mysql",
		Capacity: corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse("5Gi"),
		},
		AccessModes: []corev1.PersistentVolumeAccessMode{
			// 该卷可以被单个节点以读/写模式挂载
			corev1.ReadWriteOnce,
		},

		PersistentVolumeSource: corev1.PersistentVolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/tmp/data",
			},
		},
		PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
	}
}
