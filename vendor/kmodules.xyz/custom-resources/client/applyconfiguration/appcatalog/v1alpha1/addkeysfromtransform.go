/*
Copyright AppsCode Inc. and Contributors

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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
)

// AddKeysFromTransformApplyConfiguration represents an declarative configuration of the AddKeysFromTransform type for use
// with apply.
type AddKeysFromTransformApplyConfiguration struct {
	SecretRef *v1.LocalObjectReference `json:"secretRef,omitempty"`
}

// AddKeysFromTransformApplyConfiguration constructs an declarative configuration of the AddKeysFromTransform type for use with
// apply.
func AddKeysFromTransform() *AddKeysFromTransformApplyConfiguration {
	return &AddKeysFromTransformApplyConfiguration{}
}

// WithSecretRef sets the SecretRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SecretRef field is set to the value of the last call.
func (b *AddKeysFromTransformApplyConfiguration) WithSecretRef(value v1.LocalObjectReference) *AddKeysFromTransformApplyConfiguration {
	b.SecretRef = &value
	return b
}
