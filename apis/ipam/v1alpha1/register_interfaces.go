/*
Copyright 2021 NDD.

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
	"github.com/yndd/ndd-runtime/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ RrList = &RegisterList{}

// +k8s:deepcopy-gen=false
type RrList interface {
	client.ObjectList

	GetRegisters() []Rr
}

func (x *RegisterList) GetRegisters() []Rr {
	registers := make([]Rr, len(x.Items))
	for i, r := range x.Items {
		r := r // Pin range variable so we can take its address.
		registers[i] = &r
	}
	return registers
}

var _ Rr = &Register{}

// +k8s:deepcopy-gen=false
type Rr interface {
	resource.Object
	resource.Conditioned

	GetCondition(ct nddv1.ConditionKind) nddv1.Condition
	SetConditions(c ...nddv1.Condition)
	GetOrganization() string
	GetDeployment() string
	GetAvailabilityZone() string
	GetIpamName() string
	GetNetworkInstanceName() string
	GetIpPrefix() string
	//GetPrefixLength() *uint32
	GetAddressFamily() string
	GetSourceTag() map[string]string
	GetSelector() map[string]string
	SetIpPrefix(p string)
	HasIpPrefix() (string, bool)
	SetOrganization(string)
	SetDeployment(string)
	SetAvailabilityZone(s string)
	SetIpamName(string)
	SetNetworkInstanceName(string)
}

// GetCondition of this Network Node.
func (x *Register) GetCondition(ct nddv1.ConditionKind) nddv1.Condition {
	return x.Status.GetCondition(ct)
}

// SetConditions of the Network Node.
func (x *Register) SetConditions(c ...nddv1.Condition) {
	x.Status.SetConditions(c...)
}

func (x *Register) GetOrganization() string {
	return x.Spec.GetOrganization()
}

func (x *Register) GetDeployment() string {
	return x.Spec.GetOrganization()
}

func (x *Register) GetAvailabilityZone() string {
	return x.Spec.GetAvailabilityZone()
}

func (x *Register) GetIpamName() string {
	if reflect.ValueOf(x.Spec.RegistryName).IsZero() {
		return ""
	}
	return *x.Spec.RegistryName
}

func (x *Register) GetNetworkInstanceName() string {
	if reflect.ValueOf(x.Spec.NetworkInstanceName).IsZero() {
		return ""
	}
	return *x.Spec.NetworkInstanceName
}

func (x *Register) GetIpPrefix() string {
	if reflect.ValueOf(x.Spec.Register.IpPrefix).IsZero() {
		return ""
	}
	return *x.Spec.Register.IpPrefix
}

func (x *Register) GetAddressFamily() string {
	if reflect.ValueOf(x.Spec.Register.AddressFamily).IsZero() {
		return "ipv4"
	}
	return *x.Spec.Register.AddressFamily
}

func (x *Register) GetSourceTag() map[string]string {
	s := make(map[string]string)
	if reflect.ValueOf(x.Spec.Register.SourceTag).IsZero() {
		return s
	}
	for _, tag := range x.Spec.Register.SourceTag {
		s[*tag.Key] = *tag.Value
	}
	return s
}

func (x *Register) GetSelector() map[string]string {
	s := make(map[string]string)
	if reflect.ValueOf(x.Spec.Register.Selector).IsZero() {
		return s
	}
	for _, tag := range x.Spec.Register.Selector {
		s[*tag.Key] = *tag.Value
	}
	return s
}

func (x *Register) SetIpPrefix(p string) {
	x.Status = RegisterStatus{
		Register: &NddrIpamRegister{
			State: &NddrRegisterState{
				IpPrefix: &p,
			},
		},
	}
}

func (x *Register) HasIpPrefix() (string, bool) {
	if x.Status.Register != nil && x.Status.Register.State != nil && x.Status.Register.State.IpPrefix != nil {
		return *x.Status.Register.State.IpPrefix, true
	}
	return "", false

}

func (x *Register) SetOrganization(s string) {
	x.Status.SetOrganization(s)
}

func (x *Register) SetDeployment(s string) {
	x.Status.SetDeployment(s)
}

func (x *Register) SetAvailabilityZone(s string) {
	x.Status.SetAvailabilityZone(s)
}

func (x *Register) SetIpamName(s string) {
	x.Status.RegistryName = &s
}

func (x *Register) SetNetworkInstanceName(s string) {
	x.Status.NetworkInstanceName = &s
}
