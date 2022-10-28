package mysql

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	mysqlPV = &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mysql-pv-column",
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/mnt/data",
				},
			},
		},
	}
	mysqlPVC = &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mysql-pv-claim",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
	mysqlSVC = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mysql",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Port: 3306},
			},
			Selector: map[string]string{
				"app": "mysql",
			},
			ClusterIP: "None",
		},
	}
	mysqlConfigMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mysql-cfm",
		},
		Data: map[string]string{
			"my.cnf": `# Apply this config only on the mysql deployment.
[mysqld]
user         = root
# Accept connections from any IP address
bind-address = 0.0.0.0
`,
			"start-mysql.sh": `#!/bin/bash
function init_mysql_network_privilege() {
  sleep 3
  while true; do
    mysql -u "root" -h 127.0.0.1 -P 3306 -p123456 -E -e "update mysql.user set host='%' where user='root';flush privileges;"
    if [ $? -ne 0 ]; then
      echo "failed to update mysql user, will sleep for 1s"
    else
      echo "succeed to update mysql user"
      break
    fi
    sleep 1;
  done;
}
init_mysql_network_privilege &
/entrypoint.sh mysqld
`,
		},
	}
	mysqlDeployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mysql",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "mysql",
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "mysql",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql/mysql-server:8.0.23-aarch64",
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_ROOT_PASSWORD",
									Value: "123456",
								},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 3306, Name: "mysql"},
							},
							Command: []string{"bin/sh"},
							Args:    []string{"/etc/start-mysql.sh"},
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
								{
									Name:      "config-map",
									MountPath: "/etc/start-mysql.sh",
									SubPath:   "start-mysql.sh",
									ReadOnly:  false,
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
					Tolerations: []corev1.Toleration{
						{
							Key:      "node-role.kubernetes.io/master",
							Operator: corev1.TolerationOpEqual,
							Effect:   corev1.TaintEffectNoSchedule,
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
										Name: "mysql-cfm",
									},
									Items: []corev1.KeyToPath{
										{
											Key:  "my.cnf",
											Path: "my.cnf",
										},
										{
											Key:  "start-mysql.sh",
											Path: "start-mysql.sh",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
)

func CreateMySQLNodeCR() []client.Object {
	return []client.Object{mysqlPVC, mysqlPV, mysqlSVC, mysqlConfigMap, mysqlDeployment}
}
