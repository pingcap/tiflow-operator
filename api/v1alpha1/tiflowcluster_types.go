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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TiflowClusterSpec defines the desired state of TiflowCluster
type TiflowClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Version              string         `json:"version"`
	ConfigUpdateStrategy string         `json:"configUpdateStrategy"`
	Master               TiflowMaster   `json:"master"`
	Executor             TiflowExecutor `json:"executor"`
}

// TiflowMaster defines the desired state of tiflow master
type TiflowMaster struct {
	BaseImage        string   `json:"baseImage"`
	MaxFailoverCount int      `json:"maxFailoverCount"`
	Replicas         int      `json:"replicas"`
	Config           []string `json:"config"`
}

// TiflowExecutor defines the desired state of tiflow executor
type TiflowExecutor struct {
	BaseImage       string   `json:"baseImage"`
	PvReclaimPolicy string   `json:"pvReclaimPolicy"`
	Stateful        bool     `json:"stateful"`
	Requests        []string `json:"requests"`
	Replicas        int      `json:"replicas"`
	Config          []string `json:"config"`
}

// TiflowClusterStatus defines the observed state of TiflowCluster
type TiflowClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
