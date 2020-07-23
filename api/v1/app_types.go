/*
Copyright 2020 redwarn.

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

package v1

import (
	"github.com/containous/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Ports struct {
	Name          string `json:"name,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
	ServicePort   int32  `json:"servicePort,omitempty"`
	ContainerPort int32  `json:"containerPort,omitempty"`
}
type Resource struct {
	LimitCpu      string `json:"limitCpu,omitempty"`
	LimitMemory   string `json:"limitMemory,omitempty"`
	RequestCpu    string `json:"requestCpu,omitempty"`
	RequestMemory string `json:"requestMemory,omitempty"`
}

// AppSpec defines the desired state of App
type AppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	
	Name                string                   `json:"name"`
	Unit                string                   `json:"unit,omitempty"`
	Image               string                   `json:"image,omitempty"`
	Tag                 string                   `json:"tag,omitempty"`
	Replicas            int32                    `json:"replicas,omitempty"`
	Health              string                   `json:"health,omitempty"`
	Metrics             string                   `json:"metrics,omitempty"`
	Domain              string                   `json:"domain,omitempty"`
	Path                string                   `json:"path,omitempty"`
	PostStart           string                   `json:"postStart,omitempty"`
	PreStop             string                   `json:"preStop,omitempty"`
	Env                 map[string]string        `json:"env,omitempty"`
	Resource            *Resource                `json:"resource,omitempty"`
	Ports               []Ports                  `json:"ports,omitempty"`
	InitialDelaySeconds int32                    `json:"initialDelaySeconds,omitempty"`
	Middlewares         []v1alpha1.MiddlewareRef `json:"middlewares,omitempty"`
}

// AppStatus defines the observed state of App
type AppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Replicas            int32  `json:"replicas,omitempty"`
	// ReadyReplicas       int32  `json:"readyReplicas,omitempty"`
	// UpdatedReplicas     int32  `json:"updatedReplicas,omitempty"`
	// AvailableReplicas   int32  `json:"availableReplicas,omitempty"`
	// UnavailableReplicas int32  `json:"unavailableReplicas,omitempty"`
	// Status              string `json:"status,omitempty"` // running stoping
	// Ready bool `json:"ready,omitempty"`
}

// +kubebuilder:object:root=true
// App is the Schema for the apps API
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	
	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AppList contains a list of App
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

func init() {
	SchemeBuilder.Register(&App{}, &AppList{})
}
