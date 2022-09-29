package component

import (
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tiflow-operator/api/label"
	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

const (
	defaultHostNetwork = false
)

// ComponentAccessor is the interface to access component details, which respects the cluster-level properties
// and component-level overrides
// +output:none
type ComponentAccessor interface {
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
	DnsPolicy() corev1.DNSPolicy
	ConfigUpdateStrategy() v1alpha1.ConfigUpdateStrategy
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
	TopologySpreadConstraints() []corev1.TopologySpreadConstraint
}

// Component defines component identity of all components
type Component int

const (
	ComponentTiflowMaster Component = iota
	ComponentTiflowExecutor
)

// +output:none
type componentAccessorImpl struct {
	component Component
	name      string

	imagePullPolicy           corev1.PullPolicy
	imagePullSecrets          []corev1.LocalObjectReference
	hostNetwork               *bool
	affinity                  *corev1.Affinity
	priorityClassName         *string
	clusterNodeSelector       map[string]string
	clusterAnnotations        map[string]string
	clusterLabels             map[string]string
	tolerations               []corev1.Toleration
	dnsConfig                 *corev1.PodDNSConfig
	dnsPolicy                 corev1.DNSPolicy
	configUpdateStrategy      v1alpha1.ConfigUpdateStrategy
	statefulSetUpdateStrategy apps.StatefulSetUpdateStrategyType
	podManagementPolicy       apps.PodManagementPolicyType
	podSecurityContext        *corev1.PodSecurityContext

	// ComponentSpec is the Component Spec
	ComponentSpec *v1alpha1.ComponentSpec
}

func (a *componentAccessorImpl) StatefulSetUpdateStrategy() apps.StatefulSetUpdateStrategyType {
	if a.ComponentSpec == nil || len(a.ComponentSpec.StatefulSetUpdateStrategy) == 0 {
		if len(a.statefulSetUpdateStrategy) == 0 {
			return apps.RollingUpdateStatefulSetStrategyType
		}
		return a.statefulSetUpdateStrategy
	}
	return a.ComponentSpec.StatefulSetUpdateStrategy
}

func (a *componentAccessorImpl) PodManagementPolicy() apps.PodManagementPolicyType {
	policy := apps.ParallelPodManagement
	if a.ComponentSpec != nil && len(a.ComponentSpec.PodManagementPolicy) != 0 {
		policy = a.ComponentSpec.PodManagementPolicy
	} else if len(a.podManagementPolicy) != 0 {
		policy = a.podManagementPolicy
	}

	// unified podManagementPolicy check to avoid check everywhere
	if policy == apps.OrderedReadyPodManagement {
		return apps.OrderedReadyPodManagement
	}
	return apps.ParallelPodManagement
}

func (a *componentAccessorImpl) PodSecurityContext() *corev1.PodSecurityContext {
	if a.ComponentSpec == nil || a.ComponentSpec.PodSecurityContext == nil {
		return a.podSecurityContext
	}
	return a.ComponentSpec.PodSecurityContext
}

func (a *componentAccessorImpl) ImagePullPolicy() corev1.PullPolicy {
	if a.ComponentSpec == nil || a.ComponentSpec.ImagePullPolicy == nil {
		return a.imagePullPolicy
	}
	return *a.ComponentSpec.ImagePullPolicy
}

func (a *componentAccessorImpl) ImagePullSecrets() []corev1.LocalObjectReference {
	if a.ComponentSpec == nil || len(a.ComponentSpec.ImagePullSecrets) == 0 {
		return a.imagePullSecrets
	}
	return a.ComponentSpec.ImagePullSecrets
}

func (a *componentAccessorImpl) HostNetwork() bool {
	if a.ComponentSpec == nil || a.ComponentSpec.HostNetwork == nil {
		if a.hostNetwork == nil {
			return defaultHostNetwork
		}
		return *a.hostNetwork
	}
	return *a.ComponentSpec.HostNetwork
}

func (a *componentAccessorImpl) Affinity() *corev1.Affinity {
	if a.ComponentSpec == nil || a.ComponentSpec.Affinity == nil {
		return a.affinity
	}
	return a.ComponentSpec.Affinity
}

func (a *componentAccessorImpl) PriorityClassName() *string {
	return a.priorityClassName
}

func (a *componentAccessorImpl) NodeSelector() map[string]string {
	sel := map[string]string{}
	for k, v := range a.clusterNodeSelector {
		sel[k] = v
	}
	if a.ComponentSpec != nil {
		for k, v := range a.ComponentSpec.NodeSelector {
			sel[k] = v
		}
	}
	return sel
}

func (a *componentAccessorImpl) Labels() map[string]string {
	l := map[string]string{}
	for k, v := range a.clusterLabels {
		l[k] = v
	}
	if a.ComponentSpec != nil {
		for k, v := range a.ComponentSpec.Labels {
			l[k] = v
		}
	}
	return l
}

func (a *componentAccessorImpl) Annotations() map[string]string {
	anno := map[string]string{}
	for k, v := range a.clusterAnnotations {
		anno[k] = v
	}
	if a.ComponentSpec != nil {
		for k, v := range a.ComponentSpec.Annotations {
			anno[k] = v
		}
	}
	return anno
}

func (a *componentAccessorImpl) Tolerations() []corev1.Toleration {
	if a.ComponentSpec == nil || len(a.ComponentSpec.Tolerations) == 0 {
		return a.tolerations
	}
	return a.ComponentSpec.Tolerations
}

func (a *componentAccessorImpl) DnsPolicy() corev1.DNSPolicy {
	if a.ComponentSpec != nil && a.ComponentSpec.DNSPolicy != "" {
		return a.ComponentSpec.DNSPolicy
	}

	if a.dnsPolicy != "" {
		return a.dnsPolicy
	}

	return corev1.DNSClusterFirst // same as kubernetes default
}

func (a *componentAccessorImpl) DNSConfig() *corev1.PodDNSConfig {
	if a.ComponentSpec == nil || a.ComponentSpec.DNSConfig == nil {
		return a.dnsConfig
	}
	return a.ComponentSpec.DNSConfig
}

func (a *componentAccessorImpl) ConfigUpdateStrategy() v1alpha1.ConfigUpdateStrategy {
	// defaulting logic will set a default value for configUpdateStrategy field, but if the
	// object is created in early version without this field being set, we should set a safe default
	if a.ComponentSpec == nil || a.ComponentSpec.ConfigUpdateStrategy == nil {
		if a.configUpdateStrategy != "" {
			return a.configUpdateStrategy
		}
		return v1alpha1.ConfigUpdateStrategyInPlace
	}
	if *a.ComponentSpec.ConfigUpdateStrategy == "" {
		return v1alpha1.ConfigUpdateStrategyInPlace
	}
	return *a.ComponentSpec.ConfigUpdateStrategy
}

func (a *componentAccessorImpl) BuildPodSpec() corev1.PodSpec {
	spec := corev1.PodSpec{
		Affinity:                  a.Affinity(),
		NodeSelector:              a.NodeSelector(),
		RestartPolicy:             corev1.RestartPolicyAlways,
		Tolerations:               a.Tolerations(),
		SecurityContext:           a.PodSecurityContext(),
		TopologySpreadConstraints: a.TopologySpreadConstraints(),
		DNSPolicy:                 a.DnsPolicy(),
		DNSConfig:                 a.DNSConfig(),
	}
	if a.PriorityClassName() != nil {
		spec.PriorityClassName = *a.PriorityClassName()
	}
	if a.ImagePullSecrets() != nil {
		spec.ImagePullSecrets = a.ImagePullSecrets()
	}
	if a.TerminationGracePeriodSeconds() != nil {
		spec.TerminationGracePeriodSeconds = a.TerminationGracePeriodSeconds()
	}
	return spec
}

func (a *componentAccessorImpl) Env() []corev1.EnvVar {
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.Env
}

func (a *componentAccessorImpl) EnvFrom() []corev1.EnvFromSource {
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.EnvFrom
}

func (a *componentAccessorImpl) InitContainers() []corev1.Container {
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.InitContainers
}

func (a *componentAccessorImpl) AdditionalContainers() []corev1.Container {
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.AdditionalContainers
}

func (a *componentAccessorImpl) AdditionalVolumes() []corev1.Volume {
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.AdditionalVolumes
}

func (a *componentAccessorImpl) AdditionalVolumeMounts() []corev1.VolumeMount {
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.AdditionalVolumeMounts
}

func (a *componentAccessorImpl) TerminationGracePeriodSeconds() *int64 {
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.TerminationGracePeriodSeconds
}

func (a *componentAccessorImpl) TopologySpreadConstraints() []corev1.TopologySpreadConstraint {
	var tscs []corev1.TopologySpreadConstraint
	if a.ComponentSpec != nil && len(a.ComponentSpec.TopologySpreadConstraints) > 0 {
		tscs = a.ComponentSpec.TopologySpreadConstraints
	}

	if len(tscs) == 0 {
		return nil
	}

	ptscs := make([]corev1.TopologySpreadConstraint, 0, len(tscs))
	for _, tsc := range tscs {
		ptsc := corev1.TopologySpreadConstraint{
			MaxSkew:           1,
			TopologyKey:       tsc.TopologyKey,
			WhenUnsatisfiable: corev1.DoNotSchedule,
		}
		componentLabelVal := getComponentLabelValue(a.component)
		l := label.New().Component(componentLabelVal).Instance(a.name)
		ptsc.LabelSelector = &metav1.LabelSelector{
			MatchLabels: l,
		}
		ptscs = append(ptscs, ptsc)
	}
	return ptscs
}

func getComponentLabelValue(c Component) string {
	switch c {
	case ComponentTiflowMaster:
		return label.TiflowMasterLabelVal
	case ComponentTiflowExecutor:
		return label.TiflowExecutorLabelVal
	}
	return ""
}

func buildTiflowClusterComponentAccessor(c Component, tc *v1alpha1.TiflowCluster, componentSpec *v1alpha1.ComponentSpec) ComponentAccessor {
	spec := &tc.Spec
	return &componentAccessorImpl{
		name:                      tc.Name,
		component:                 c,
		imagePullPolicy:           spec.ImagePullPolicy,
		imagePullSecrets:          spec.ImagePullSecrets,
		hostNetwork:               spec.HostNetwork,
		affinity:                  spec.Affinity,
		priorityClassName:         spec.PriorityClassName,
		clusterNodeSelector:       spec.NodeSelector,
		clusterLabels:             spec.Labels,
		clusterAnnotations:        spec.Annotations,
		tolerations:               spec.Tolerations,
		dnsConfig:                 spec.DNSConfig,
		dnsPolicy:                 spec.DNSPolicy,
		configUpdateStrategy:      spec.ConfigUpdateStrategy,
		statefulSetUpdateStrategy: spec.StatefulSetUpdateStrategy,
		podManagementPolicy:       spec.PodManagementPolicy,
		podSecurityContext:        spec.PodSecurityContext,

		ComponentSpec: componentSpec,
	}
}

func BuildMasterSpec(tc *v1alpha1.TiflowCluster) ComponentAccessor {
	var spec *v1alpha1.ComponentSpec
	if tc.Spec.Master != nil {
		spec = &tc.Spec.Master.ComponentSpec
	}
	return buildTiflowClusterComponentAccessor(ComponentTiflowMaster, tc, spec)
}

func BuildExecutorSpec(tc *v1alpha1.TiflowCluster) ComponentAccessor {
	var spec *v1alpha1.ComponentSpec
	if tc.Spec.Executor != nil {
		spec = &tc.Spec.Executor.ComponentSpec
	}

	return buildTiflowClusterComponentAccessor(ComponentTiflowExecutor, tc, spec)
}
