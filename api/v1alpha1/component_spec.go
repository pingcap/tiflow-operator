package v1alpha1

import (
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type ComponentAccessor interface {
	MemberType() MemberType
	ImagePullPolicy() corev1.PullPolicy
	ImagePullSecrets() []corev1.LocalObjectReference
	HostNetwork() bool
	Affinity() *corev1.Affinity
	PriorityClassName() *string
	NodeSelector() map[string]string
	Labels() map[string]string
	Annotations() map[string]string
	Tolerations() []corev1.Toleration
	PodSecurityContext() *corev1.PodSecurityContext
	SchedulerName() string
	DnsPolicy() corev1.DNSPolicy
	ConfigUpdateStrategy() ConfigUpdateStrategy
	BuildPodSpec() corev1.PodSpec
	Env() []corev1.EnvVar
	EnvFrom() []corev1.EnvFromSource
	AdditionalContainers() []corev1.Container
	InitContainers() []corev1.Container
	AdditionalVolumes() []corev1.Volume
	AdditionalVolumeMounts() []corev1.VolumeMount
	TerminationGracePeriodSeconds() *int64
	StatefulSetUpdateStrategy() apps.StatefulSetUpdateStrategyType
	PodManagementPolicy() apps.PodManagementPolicyType
}

type componentAccessorImpl struct {
	component MemberType
	name      string
	kind      string

	imagePullPolicy           corev1.PullPolicy
	imagePullSecrets          []corev1.LocalObjectReference
	hostNetwork               *bool
	affinity                  *corev1.Affinity
	priorityClassName         *string
	schedulerName             string
	clusterNodeSelector       map[string]string
	clusterAnnotations        map[string]string
	clusterLabels             map[string]string
	tolerations               []corev1.Toleration
	dnsConfig                 *corev1.PodDNSConfig
	dnsPolicy                 corev1.DNSPolicy
	configUpdateStrategy      ConfigUpdateStrategy
	statefulSetUpdateStrategy apps.StatefulSetUpdateStrategyType
	podManagementPolicy       apps.PodManagementPolicyType
	podSecurityContext        *corev1.PodSecurityContext

	// ComponentSpec is the Component Spec
	ComponentSpec *ComponentSpec
}

func (a *componentAccessorImpl) ImagePullPolicy() corev1.PullPolicy {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) ImagePullSecrets() []corev1.LocalObjectReference {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) HostNetwork() bool {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) Affinity() *corev1.Affinity {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) PriorityClassName() *string {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) NodeSelector() map[string]string {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) Labels() map[string]string {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) Annotations() map[string]string {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) Tolerations() []corev1.Toleration {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) PodSecurityContext() *corev1.PodSecurityContext {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) SchedulerName() string {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) DnsPolicy() corev1.DNSPolicy {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) ConfigUpdateStrategy() ConfigUpdateStrategy {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) BuildPodSpec() corev1.PodSpec {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) Env() []corev1.EnvVar {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) EnvFrom() []corev1.EnvFromSource {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) AdditionalContainers() []corev1.Container {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) InitContainers() []corev1.Container {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) AdditionalVolumes() []corev1.Volume {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) AdditionalVolumeMounts() []corev1.VolumeMount {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) TerminationGracePeriodSeconds() *int64 {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) StatefulSetUpdateStrategy() apps.StatefulSetUpdateStrategyType {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) PodManagementPolicy() apps.PodManagementPolicyType {
	//TODO implement me
	panic("implement me")
}

func (a *componentAccessorImpl) MemberType() MemberType {
	return a.component
}

func buildTiFLowClusterComponentAccessor(c MemberType, tc *TiflowCluster, componentSpec *ComponentSpec) ComponentAccessor {
	//TODO implement me
	panic("implement me")
}
