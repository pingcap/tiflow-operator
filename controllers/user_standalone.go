package controllers

import (
	"context"
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *TiflowClusterReconciler) ReconcileUserStandalone(ctx context.Context, instance *v1alpha1.TiflowCluster, req ctrl.Request) (ctrl.Result, error) {

	logg := log.FromContext(ctx)

	var cfm corev1.ConfigMap
	cfm.Name = "etcdcfm"
	cfm.Namespace = instance.Namespace

	or, err := ctrl.CreateOrUpdate(ctx, r.Client, &cfm, func() error {
		MakeConfigMapIfNotExists(&cfm)
		return controllerutil.SetControllerReference(instance, &cfm, r.Scheme)
	})

	if err != nil {
		logg.Error(err, "create etcd configMap error")
		return ctrl.Result{}, err
	}
	logg.Info("CreateOrUpdate", "Etcd ConfigMap", or)

	var svc corev1.Service
	svc.Name = USER_STANDALONE
	svc.Namespace = instance.Namespace

	or, err = ctrl.CreateOrUpdate(ctx, r.Client, &svc, func() error {
		MakeServiceIfNotExists(instance, &svc)
		return controllerutil.SetControllerReference(instance, &svc, r.Scheme)
	})

	if err != nil {
		logg.Error(err, "create etcd service error")
		return ctrl.Result{}, err
	}

	logg.Info("CreateOrUpdate", "Etcd Service", or)

	var deploy appsv1.Deployment
	deploy.Name = USER_STANDALONE
	deploy.Namespace = instance.Namespace
	or, err = ctrl.CreateOrUpdate(ctx, r.Client, &deploy, func() error {
		MakeDeploymentIfNotExists(instance, &deploy)
		return controllerutil.SetControllerReference(instance, &deploy, r.Scheme)
	})

	if err != nil {
		logg.Error(err, "create etcd deployment error")
	}

	logg.Info("CreateOrUpdate", "Etcd Deployment", or)

	logg.Info("user standalone reconcile end", "reconcile", "success")

	return ctrl.Result{}, nil
}

func MakeConfigMapIfNotExists(cfm *corev1.ConfigMap) {
	cfm.Labels = map[string]string{
		USER_STANDALONE: "etcd",
	}

	cfm.Data = map[string]string{
		"etcd.yml": `# This is the configuration file for the etcd server.
# Human-readable name for this member.
name: 'dataflow-etcd'
# Path to the data directory.
#data-dir: '/tmp/dataflow-etcd'
# Path to the dedicated wal directory.
#wal-dir:  '/tmp/dataflow-etcd'
# Number of committed transactions to trigger a snapshot to disk.
snapshot-count: 10000
# Time (in milliseconds) of a heartbeat interval.
heartbeat-interval: 100
# Time (in milliseconds) for an election to timeout.
election-timeout: 1000
# Raise alarms when backend size exceeds the given quota. 0 means use the
# default quota.
quota-backend-bytes: 0
# List of comma separated URLs to listen on for peer traffic.
listen-peer-urls: http://0.0.0.0:2380
# List of comma separated URLs to listen on for client traffic.
listen-client-urls: http://0.0.0.0:2379
# Maximum number of snapshot files to retain (0 is unlimited).
max-snapshots: 5
# Maximum number of wal files to retain (0 is unlimited).
max-wals: 5
# Comma-separated white list of origins for CORS (cross-origin resource sharing).
cors:
# List of this member's peer URLs to advertise to the rest of the cluster.
# The URLs needed to be a comma-separated list.
initial-advertise-peer-urls: http://0.0.0.0:2380
# List of this member's client URLs to advertise to the public.
# The URLs needed to be a comma-separated list.
advertise-client-urls: http://0.0.0.0:2379
# Discovery URL used to bootstrap the cluster.
discovery:
# Valid values include 'exit', 'proxy'
discovery-fallback: 'proxy'
# HTTP proxy to use for traffic to discovery service.
discovery-proxy:
# DNS domain used to bootstrap initial cluster.
discovery-srv:
# Initial cluster configuration for bootstrapping.
initial-cluster:  'dataflow-etcd=http://0.0.0.0:2380'
# Initial cluster token for the etcd cluster during bootstrap.
initial-cluster-token: 'etcd-cluster'
# Initial cluster state ('new' or 'existing').
initial-cluster-state: 'new'
# Reject reconfiguration requests that would cause quorum loss.
strict-reconfig-check: false
# Enable runtime profiling data via HTTP server
enable-pprof: true
# Valid values include 'on', 'readonly', 'off'
proxy: 'off'
# Time (in milliseconds) an endpoint will be held in a failed state.
proxy-failure-wait: 5000
# Time (in milliseconds) of the endpoints refresh interval.
proxy-refresh-interval: 30000
# Time (in milliseconds) for a dial to timeout.
proxy-dial-timeout: 1000
# Time (in milliseconds) for a write to timeout.
proxy-write-timeout: 5000
# Time (in milliseconds) for a read to timeout.
proxy-read-timeout: 0
client-transport-security:
  # Path to the client server TLS cert file.
  cert-file:
  # Path to the client server TLS key file.
  key-file:
  # Enable client cert authentication.
  client-cert-auth: false
  # Path to the client server TLS trusted CA cert file.
  trusted-ca-file:
  # Client TLS using generated certificates
  auto-tls: false
peer-transport-security:
  # Path to the peer server TLS cert file.
  cert-file:
  # Path to the peer server TLS key file.
  key-file:
  # Enable peer client cert authentication.
  client-cert-auth: false
  # Path to the peer server TLS trusted CA cert file.
  trusted-ca-file:
  # Peer TLS using generated certificates.
  auto-tls: false
# The validity period of the self-signed certificate, the unit is year.
self-signed-cert-validity: 1
# Enable debug-level logging for etcd.
log-level: debug
logger: zap
# Specify 'stdout' or 'stderr' to skip journald logging even when running under systemd.
log-outputs: [stderr]
# Force to create a new one member cluster.
force-new-cluster: false`,
	}
}

func MakeServiceIfNotExists(de *v1alpha1.TiflowCluster, svc *corev1.Service) {

	svc.Labels = map[string]string{
		USER_STANDALONE: "etcd-standalone",
	}

	svc.Spec = corev1.ServiceSpec{
		ClusterIP: corev1.ClusterIPNone,
		Selector: map[string]string{
			"app": USER_STANDALONE,
		},
		Ports: []corev1.ServicePort{
			{
				Name:     "peer",
				Port:     2380,
				Protocol: corev1.ProtocolTCP,
			},
			{
				Name:     "client",
				Port:     2379,
				Protocol: corev1.ProtocolTCP,
			},
		},
	}
}

func MakeDeploymentIfNotExists(de *v1alpha1.TiflowCluster, deploy *appsv1.Deployment) {
	deploy.Labels = map[string]string{
		USER_STANDALONE: "etcd-standalone",
	}

	var size int32
	size = 1

	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: &size,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": USER_STANDALONE,
			},
		},
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": USER_STANDALONE,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "user-etcd-standalone",
						Image:           "quay.io/coreos/etcd",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command: []string{
							"etcd", "--config-file",
							"/mnt/config-map/etcd.yml",
						},
						Ports: []corev1.ContainerPort{
							{
								Name:          "peer",
								ContainerPort: 2380,
								Protocol:      corev1.ProtocolTCP,
							},
							{
								Name:          "client",
								ContainerPort: 2379,
								Protocol:      corev1.ProtocolTCP,
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "config-map",
								MountPath: "/mnt/config-map",
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "config-map",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "etcdcfm",
								},
							},
						},
					},
					{
						Name: "conf-etcd",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{
								Medium: corev1.StorageMediumDefault,
							},
						},
					},
				},
			},
		},
	}
}
