package utils

import (
	"context"
	"encoding/json"
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tiflow-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// LastAppliedConfigAnnotation is annotation key of last applied configuration
	LastAppliedConfigAnnotation = "pingcap.com/last-applied-configuration"
)

// UpdateStatefulSetWithPreCheck todo: impl this logic
func UpdateStatefulSetWithPreCheck(tc *v1alpha1.TiflowCluster, resaon string, newSts, oldSts *apps.StatefulSet) error {
	return nil
}

// SetStatefulSetLastAppliedConfigAnnotation set last applied config to Statefulset's annotation
func SetStatefulSetLastAppliedConfigAnnotation(set *apps.StatefulSet) error {
	setApply, err := util.Encode(set.Spec)
	if err != nil {
		return err
	}
	if set.Annotations == nil {
		set.Annotations = map[string]string{}
	}
	set.Annotations[LastAppliedConfigAnnotation] = setApply
	return nil
}

// UpdateStatefulSet is a template function to update the statefulset of components
func UpdateStatefulSet(ctx context.Context, cli client.Client, newSet, oldSet *apps.StatefulSet) error {
	isOrphan := metav1.GetControllerOf(oldSet) == nil
	if newSet.Annotations == nil {
		newSet.Annotations = map[string]string{}
	}
	if oldSet.Annotations == nil {
		oldSet.Annotations = map[string]string{}
	}

	// Check if an upgrade is needed.
	// If not, early return.
	if StatefulSetEqual(*newSet, *oldSet) && !isOrphan {
		return nil
	}

	set := *oldSet

	// update specs for sts
	*set.Spec.Replicas = *newSet.Spec.Replicas
	set.Spec.UpdateStrategy = newSet.Spec.UpdateStrategy
	set.Labels = newSet.Labels
	set.Annotations = newSet.Annotations
	set.Spec.Template = newSet.Spec.Template
	if isOrphan {
		set.OwnerReferences = newSet.OwnerReferences
	}

	var podConfig string
	var hasPodConfig bool
	if oldSet.Spec.Template.Annotations != nil {
		podConfig, hasPodConfig = oldSet.Spec.Template.Annotations[LastAppliedConfigAnnotation]
	}
	if hasPodConfig {
		if set.Spec.Template.Annotations == nil {
			set.Spec.Template.Annotations = map[string]string{}
		}
		set.Spec.Template.Annotations[LastAppliedConfigAnnotation] = podConfig
	}
	v, ok := oldSet.Annotations[label.AnnStsLastSyncTimestamp]
	if ok {
		set.Annotations[label.AnnStsLastSyncTimestamp] = v
	}

	err := SetStatefulSetLastAppliedConfigAnnotation(&set)
	if err != nil {
		return err
	}

	// commit to k8s
	err = cli.Update(ctx, &set)
	return err
}

// SetUpgradePartition set statefulSet's rolling update partition
func SetUpgradePartition(set *apps.StatefulSet, upgradeOrdinal int32) {
	set.Spec.UpdateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{Partition: &upgradeOrdinal}
	klog.Infof("set %s/%s partition to %d", set.GetNamespace(), set.GetName(), upgradeOrdinal)
}

// StatefulSetEqual compares the new Statefulset's spec with old Statefulset's last applied config
func StatefulSetEqual(new apps.StatefulSet, old apps.StatefulSet) bool {
	// The annotations in old sts may include LastAppliedConfigAnnotation
	tmpAnno := map[string]string{}
	for k, v := range old.Annotations {
		if k != LastAppliedConfigAnnotation && k != label.AnnStsLastSyncTimestamp {
			tmpAnno[k] = v
		}
	}
	if !apiequality.Semantic.DeepEqual(new.Annotations, tmpAnno) {
		return false
	}
	oldConfig := apps.StatefulSetSpec{}
	if lastAppliedConfig, ok := old.Annotations[LastAppliedConfigAnnotation]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldConfig)
		if err != nil {
			klog.Errorf("unmarshal Statefulset: [%s/%s]'s applied config failed,error: %v", old.GetNamespace(), old.GetName(), err)
			return false
		}
		// oldConfig.Template.Annotations may include LastAppliedConfigAnnotation to keep backward compatiability
		// Please check detail in https://github.com/pingcap/tidb-operator/pull/1489
		tmpTemplate := oldConfig.Template.DeepCopy()
		delete(tmpTemplate.Annotations, LastAppliedConfigAnnotation)
		return apiequality.Semantic.DeepEqual(oldConfig.Replicas, new.Spec.Replicas) &&
			apiequality.Semantic.DeepEqual(*tmpTemplate, new.Spec.Template) &&
			apiequality.Semantic.DeepEqual(oldConfig.UpdateStrategy, new.Spec.UpdateStrategy)
	}
	return false
}
