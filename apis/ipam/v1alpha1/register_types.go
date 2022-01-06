/*
Copyright 2021 NDDO.

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
	"reflect"

	nddv1 "github.com/yndd/ndd-runtime/apis/common/v1"
	nddov1 "github.com/yndd/nddo-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NddrIpamRegister struct
type NddrIpamRegister struct {
	IpamRegister `json:",inline"`
	State        *NddrRegisterState `json:"state,omitempty"`
}

// NddrRegisterState struct
type NddrRegisterState struct {
	IpPrefix *string `json:"ip-prefix,omitempty"`
	//ExpiryTime *string `json:"expiry-time,omitempty"`
}

// IpamRegister struct
type IpamRegister struct {
	// +kubebuilder:validation:Enum=`ipv4`;`ipv6`
	// +kubebuilder:default:="ipv4"
	AddressFamily *string `json:"address-family,omitempty"`
	IpPrefix      *string `json:"ip-prefix,omitempty"`
	// kubebuilder:validation:Minimum=0
	// kubebuilder:validation:Maximum=128
	//PrefixLength *uint32       `json:"prefix-length,omitempty"`
	Selector  []*nddov1.Tag `json:"selector,omitempty"`
	SourceTag []*nddov1.Tag `json:"source-tag,omitempty"`
}

// A RegisterSpec defines the desired state of a Register.
type RegisterSpec struct {
	//nddov1.OdaInfo      `json:",inline"`
	//RegistryName        *string       `json:"registry-name,omitempty"`
	//NetworkInstanceName *string       `json:"network-instance-name,omitempty"`
	Register *IpamRegister `json:"register,omitempty"`
}

// A RegisterStatus represents the observed state of a Register.
type RegisterStatus struct {
	nddv1.ConditionedStatus `json:",inline"`
	nddov1.OdaInfo          `json:",inline"`
	RegistryName            *string           `json:"registry-name,omitempty"`
	NetworkInstanceName     *string           `json:"network-instance-name,omitempty"`
	Register                *NddrIpamRegister `json:"register,omitempty"`
}

// +kubebuilder:object:root=true

// Register is the Schema for the Register API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SYNC",type="string",JSONPath=".status.conditions[?(@.kind=='Synced')].status"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.conditions[?(@.kind=='Ready')].status"
// +kubebuilder:printcolumn:name="ORG",type="string",JSONPath=".status.oda[?(@.key=='organization')].value"
// +kubebuilder:printcolumn:name="DEP",type="string",JSONPath=".status.oda[?(@.key=='deployment')].value"
// +kubebuilder:printcolumn:name="AZ",type="string",JSONPath=".status.oda[?(@.key=='availability-zone')].value"
// +kubebuilder:printcolumn:name="REGISTRY",type="string",JSONPath=".status.registry-name"
// +kubebuilder:printcolumn:name="IPPREFIX",type="string",JSONPath=".status.register.state.ip-prefix",description="assigned IP Prefix"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
type Register struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegisterSpec   `json:"spec,omitempty"`
	Status RegisterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RegisterList contains a list of Registers
type RegisterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Register `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Register{}, &RegisterList{})
}

// Register type metadata.
var (
	RegisterKindKind         = reflect.TypeOf(Register{}).Name()
	RegisterGroupKind        = schema.GroupKind{Group: Group, Kind: RegisterKindKind}.String()
	RegisterKindAPIVersion   = RegisterKindKind + "." + GroupVersion.String()
	RegisterGroupVersionKind = GroupVersion.WithKind(RegisterKindKind)
)
