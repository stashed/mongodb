/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Free Trial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Free-Trial-1.0.0.md

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

const (
	ResourceKindStashMongodb = "StashMongodb"
	ResourceStashMongodb     = "stashmongodb"
	ResourceStashMongodbs    = "stashmongodbs"
)

// StashMongodb defines the schama for Stash MongoDB Installer.

// +genclient
// +genclient:skipVerbs=updateStatus
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=stashmongodbs,singular=stashmongodb,categories={stash,appscode}
type StashMongodb struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              StashMongodbSpec `json:"spec,omitempty"`
}

// StashMongodbSpec is the schema for Stash MongoDB values file
type StashMongodbSpec struct {
	// +optional
	NameOverride string `json:"nameOverride"`
	// +optional
	FullnameOverride string         `json:"fullnameOverride"`
	Image            ImageRef       `json:"image"`
	Backup           MongoDBBackup  `json:"backup"`
	Restore          MongoDBRestore `json:"restore"`
	MaxConcurrency   int32          `json:"maxConcurrency"`
	WaitTimeout      int64          `json:"waitTimeout"`
}

type ImageRef struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

type MongoDBBackup struct {
	// +optional
	Args string `json:"args"`
}

type MongoDBRestore struct {
	// +optional
	Args string `json:"args"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StashMongodbList is a list of StashMongodbs
type StashMongodbList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// Items is a list of StashMongodb CRD objects
	Items []StashMongodb `json:"items,omitempty"`
}
