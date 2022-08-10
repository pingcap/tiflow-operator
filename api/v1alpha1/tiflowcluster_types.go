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

package v1alpha1

import (
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type MemberType string

const (
	TiFlowMasterMemberType   MemberType = "tiflow-master"
	TiFlowExecutorMemberType MemberType = "tiflow-executor"
)

// ConfigUpdateStrategy represents the strategy to update configuration
type ConfigUpdateStrategy string

const (
	// ConfigUpdateStrategyInPlace update the configmap without changing the name
	ConfigUpdateStrategyInPlace ConfigUpdateStrategy = "InPlace"
	// ConfigUpdateStrategyRollingUpdate generate different configmap on configuration update and
	// try to rolling-update the pod controller (e.g. statefulset) to apply updates.
	ConfigUpdateStrategyRollingUpdate ConfigUpdateStrategy = "RollingUpdate"
)

// ComponentSpec is the base spec of each component, the fields should always accessed by the Basic<Component>Spec() method to respect the cluster-level properties
// +k8s:openapi-gen=true
type ComponentSpec struct {
	// BaseImage of the component. Override the cluster-level baseImage if present
	BaseImage string `json:"baseImage,omitempty"`

	// Version of the component. Override the cluster-level version if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	Version *string `json:"version,omitempty"`

	// ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
	// Optional: Defaults to cluster-level setting
	// +optional
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Whether Hostnetwork of the component is enabled. Override the cluster-level setting if present
	// Optional: Defaults to cluster-level setting
	// +optional
	HostNetwork *bool `json:"hostNetwork,omitempty"`

	// Affinity of the component. Override the cluster-level setting if present.
	// Optional: Defaults to cluster-level setting
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// NodeSelector of the component. Merged into the cluster-level nodeSelector if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Annotations for the component. Merge into the cluster-level annotations if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels for the component. Merge into the cluster-level labels if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Tolerations of the component. Override the cluster-level tolerations if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// PodSecurityContext of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// ConfigUpdateStrategy of the component. Override the cluster-level updateStrategy if present
	// Optional: Defaults to cluster-level setting
	// +optional
	ConfigUpdateStrategy *ConfigUpdateStrategy `json:"configUpdateStrategy,omitempty"`

	// List of environment variables to set in the container, like v1.Container.Env.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Extend the use scenarios for env
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// Init containers of the components
	// +optional
	InitContainers []corev1.Container `json:"initContainers,omitempty"`

	// Additional containers of the component.
	// +optional
	AdditionalContainers []corev1.Container `json:"additionalContainers,omitempty"`

	// Additional volumes of component pod.
	// +optional
	AdditionalVolumes []corev1.Volume `json:"additionalVolumes,omitempty"`

	// Additional volume mounts of component pod.
	AdditionalVolumeMounts []corev1.VolumeMount `json:"additionalVolumeMounts,omitempty"`

	// DNSConfig Specifies the DNS parameters of a pod.
	// +optional
	DNSConfig *corev1.PodDNSConfig `json:"dnsConfig,omitempty"`

	// DNSPolicy Specifies the DNSPolicy parameters of a pod.
	// +optional
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request.
	// Value must be non-negative integer. The value zero indicates delete immediately.
	// If this value is nil, the default grace period will be used instead.
	// The grace period is the duration in seconds after the processes running in the pod are sent
	// a termination signal and the time when the processes are forcibly halted with a kill signal.
	// Set this value longer than the expected cleanup time for your process.
	// Defaults to 30 seconds.
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`

	// StatefulSetUpdateStrategy indicates the StatefulSetUpdateStrategy that will be
	// employed to update Pods in the StatefulSet when a revision is made to
	// Template.
	// +optional
	StatefulSetUpdateStrategy apps.StatefulSetUpdateStrategyType `json:"statefulSetUpdateStrategy,omitempty"`

	// PodManagementPolicy of TiDB cluster StatefulSets
	// +optional
	PodManagementPolicy apps.PodManagementPolicyType `json:"podManagementPolicy,omitempty"`

	// TopologySpreadConstraints describes how a group of pods ought to spread across topology
	// domains. Scheduler will schedule pods in a way which abides by the constraints.
	// This field is is only honored by clusters that enables the EvenPodsSpread feature.
	// All topologySpreadConstraints are ANDed.
	// +optional
	// +listType=map
	// +listMapKey=topologyKey
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

// ServiceSpec specifies the service object in k8s
// +k8s:openapi-gen=true
type ServiceSpec struct {
	// Type of the real kubernetes service
	Type corev1.ServiceType `json:"type,omitempty"`

	// Additional annotations for the service
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Additional labels for the service
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// LoadBalancerIP is the loadBalancerIP of service
	// Optional: Defaults to omitted
	// +optional
	LoadBalancerIP *string `json:"loadBalancerIP,omitempty"`

	// ClusterIP is the clusterIP of service
	// +optional
	ClusterIP *string `json:"clusterIP,omitempty"`

	// PortName is the name of service port
	// +optional
	PortName *string `json:"portName,omitempty"`

	// The port that will be exposed by this service.
	//
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	NodePort *int32 `json:"port,omitempty"`

	// LoadBalancerSourceRanges is the loadBalancerSourceRanges of service
	// If specified and supported by the platform, this will restrict traffic through the cloud-provider
	// load-balancer will be restricted to the specified client IPs. This field will be ignored if the
	// cloud-provider does not support the feature."
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#aws-nlb-support
	// Optional: Defaults to omitted
	// +optional
	LoadBalancerSourceRanges []string `json:"loadBalancerSourceRanges,omitempty"`

	// ExternalTrafficPolicy of the service
	// Optional: Defaults to omitted
	// +optional
	ExternalTrafficPolicy *corev1.ServiceExternalTrafficPolicyType `json:"externalTrafficPolicy,omitempty"` // Expose the tidb cluster mysql port to MySQLNodePort
}

// MasterSpec defines the desired state of tiflow master
type MasterSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/tiflow
	// +optional
	BaseImage string `json:"baseImage"`

	// Service defines a Kubernetes service of Master cluster.
	// Optional: Defaults to `.spec.services` in favor of backward compatibility
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover.
	// Optional: Defaults to 3
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount int `json:"maxFailoverCount"`

	// Config is the Configuration of tiflow-master-servers
	// +optional
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	Config *config.GenericConfig `json:"config,omitempty"`
}

// ExecutorSpec defines the desired state of tiflow executor
type ExecutorSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/tiflow
	// +optional
	BaseImage string `json:"baseImage,omitempty"`

	// MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover.
	// Optional: Defaults to 3
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// The storageClassName of the persistent volume for tiflow-executor data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// StorageSize is the request storage size for tiflow-executor.
	// Defaults to "10Gi".
	// +optional
	StorageSize string `json:"storageSize,omitempty"`

	// Subdirectory within the volume to store tiflow-executor Data. By default, the data
	// is stored in the root directory of volume which is mounted at
	// /tmp/tiflow-executor.
	// Specifying this will change the data directory to a subdirectory, e.g.
	// /tmp/tiflow-executor/data if you set the value to "data".
	// It's dangerous to change this value for a running cluster as it will
	// upgrade your cluster to use a new storage directory.
	// Defaults to "" (volume's root).
	// +optional
	DataSubDir string `json:"dataSubDir,omitempty"`

	// Persistent volume reclaim policy applied to the PVs that consumed by TiDB cluster
	// +kubebuilder:default=Retain
	PVReclaimPolicy *corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`

	// Stateful indicates whether this executor will deal with stateful jobs preferentially.
	// If enabled, master will firstly arrange stateful jobs to this kind of executors and operator will keep this executor's data for more time.
	// +kubebuilder:default=false
	// +optional
	Stateful bool `json:"stateful"`

	// Config is the Configuration of tiflow-executor-servers
	// +optional
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	Config *config.GenericConfig `json:"config,omitempty"`
}

// TiflowClusterSpec defines the desired state of TiflowCluster
type TiflowClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Tiflow-master cluster spec
	// +optional
	Master *MasterSpec `json:"master"`

	// Tiflow-executor cluster spec
	// +optional
	Executor *ExecutorSpec `json:"executor"`

	// TiDB cluster version
	// +optional
	Version string `json:"version"`

	// ImagePullPolicy of Tiflow cluster Pods
	// +kubebuilder:default=IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// ConfigUpdateStrategy determines how the configuration change is applied to the cluster.
	// UpdateStrategyInPlace will update the ConfigMap of configuration in-place and an extra rolling-update of the
	// cluster component is needed to reload the configuration change.
	// UpdateStrategyRollingUpdate will create a new ConfigMap with the new configuration and rolling-update the
	// related components to use the new ConfigMap, that is, the new configuration will be applied automatically.
	ConfigUpdateStrategy ConfigUpdateStrategy `json:"configUpdateStrategy,omitempty"`

	// Whether enable the TLS connection between Tiflow components
	// Optional: Defaults to nil
	// +optional
	TLSCluster bool `json:"tlsCluster,omitempty"`

	// Whether Hostnetwork is enabled for Tiflow cluster Pods
	// Optional: Defaults to false
	// +optional
	HostNetwork *bool `json:"hostNetwork,omitempty"`

	// Affinity of Tiflow cluster Pods
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// PriorityClassName of Tiflow cluster Pods
	// Optional: Defaults to omitted
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// Base node selectors of Tiflow cluster Pods, components may add or override selectors upon this respectively
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Additional annotations for the dm cluster
	// Can be overrode by annotations in master spec or executor spec
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Additional labels for the tiflow cluster
	// Can be overrode by labels in master spec or executor spec
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Base tolerations of Tiflow cluster Pods, components may add more tolerations upon this respectively
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// DNSConfig Specifies the DNS parameters of a pod.
	// +optional
	DNSConfig *corev1.PodDNSConfig `json:"dnsConfig,omitempty"`

	// DNSPolicy Specifies the DNSPolicy parameters of a pod.
	// +optional
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// PodSecurityContext of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// StatefulSetUpdateStrategy of Tiflow cluster StatefulSets
	// +optional
	StatefulSetUpdateStrategy apps.StatefulSetUpdateStrategyType `json:"statefulSetUpdateStrategy,omitempty"`

	// PodManagementPolicy of Tiflow cluster StatefulSets
	// +optional
	PodManagementPolicy apps.PodManagementPolicyType `json:"podManagementPolicy,omitempty"`
}

// MemberPhase is the current state of member
type MemberPhase string

const (
	// NormalPhase represents normal state of TiDB cluster.
	NormalPhase MemberPhase = "Normal"
	// UpgradePhase represents the upgrade state of TiDB cluster.
	UpgradePhase MemberPhase = "Upgrade"
	// ScalePhase represents the scaling state of TiDB cluster.
	ScalePhase MemberPhase = "Scale"
)

// MasterStatus defines the desired state of Tiflow-master
type MasterStatus struct {
	Synced          bool                    `json:"synced,omitempty"`
	Phase           MemberPhase             `json:"phase,omitempty"`
	StatefulSet     *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
	Members         map[string]MasterMember `json:"members,omitempty"`
	Leader          MasterMember            `json:"leader,omitempty"`
	FailureMembers  map[string]MasterMember `json:"failureMembers,omitempty"`
	UnjoinedMembers map[string]MasterMember `json:"unjoinedMembers,omitempty"`
	Image           string                  `json:"image,omitempty"`
}

// MasterMember is Tiflow-master member status
type MasterMember struct {
	Name string `json:"name"`
	// member id is actually a uint64, but apimachinery's json only treats numbers as int64/float64
	// so uint64 may overflow int64 and thus convert to float64
	MemberID      string `json:"memberID"`
	PodName       string `json:"podName,omitempty"`
	ClientURL     string `json:"clientURL"`
	Health        bool   `json:"health"`
	MemberDeleted bool   `json:"memberDeleted,omitempty"`
	// Last time the health transitioned from one to another.
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// ExecutorMember is Tiflow-executor member status
type ExecutorMember struct {
	Name string `json:"name,omitempty"`
	Addr string `json:"addr,omitempty"`
	// TODO: add cpu/memory/disk usage later
	// Last time the health transitioned from one to another.
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

type ObservedStorageVolumeStatus struct {
	// BoundCount is the count of bound volumes.
	// +optional
	BoundCount int `json:"boundCount"`
	// CurrentCount is the count of volumes whose capacity is equal to `currentCapacity`.
	// +optional
	CurrentCount int `json:"currentCount"`
	// ResizedCount is the count of volumes whose capacity is equal to `resizedCapacity`.
	// +optional
	ResizedCount int `json:"resizedCount"`
	// CurrentCapacity is the current capacity of the volume.
	// If any volume is resizing, it is the capacity before resizing.
	// If all volumes are resized, it is the resized capacity and same as desired capacity.
	CurrentCapacity resource.Quantity `json:"currentCapacity"`
	// ResizedCapacity is the desired capacity of the volume.
	ResizedCapacity resource.Quantity `json:"resizedCapacity"`
}

// StorageVolumeName is the volume name which is same as `volumes.name` in Pod spec.
type StorageVolumeName string

// StorageVolumeStatus is the actual status for a storage
type StorageVolumeStatus struct {
	ObservedStorageVolumeStatus `json:",inline"`
	// Name is the volume name which is same as `volumes.name` in Pod spec.
	Name StorageVolumeName `json:"name"`
}

// ExecutorStatus defines the desired state of Tiflow-executor
type ExecutorStatus struct {
	Synced         bool                      `json:"synced,omitempty"`
	Phase          MemberPhase               `json:"phase,omitempty"`
	StatefulSet    *apps.StatefulSetStatus   `json:"statefulSet,omitempty"`
	Members        map[string]ExecutorMember `json:"members,omitempty"`
	FailureMembers map[string]ExecutorMember `json:"failureMembers,omitempty"`
	FailoverUID    types.UID                 `json:"failoverUID,omitempty"`
	Image          string                    `json:"image,omitempty"`
	// Volumes contains the status of all volumes.
	Volumes map[string]*StorageVolumeStatus `json:"volumes,omitempty"`
}

// TiflowClusterCondition is tiflow cluster condition
type TiflowClusterCondition struct {
	// Type of the condition.
	Type string `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	// +nullable
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	// +nullable
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// TiflowClusterStatus defines the observed state of TiflowCluster
type TiflowClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Master   MasterStatus   `json:"master,omitempty"`
	Executor ExecutorStatus `json:"executor,omitempty"`

	// Represents the latest available observations of a tiflow cluster's state.
	// +optional
	// +nullable
	Conditions []TiflowClusterCondition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TiflowCluster is the Schema for the tiflowclusters API
type TiflowCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiflowClusterSpec   `json:"spec,omitempty"`
	Status TiflowClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TiflowClusterList contains a list of TiflowCluster
type TiflowClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TiflowCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TiflowCluster{}, &TiflowClusterList{})
}
