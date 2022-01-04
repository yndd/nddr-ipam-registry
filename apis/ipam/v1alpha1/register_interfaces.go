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
	"strings"

	nddv1 "github.com/yndd/ndd-runtime/apis/common/v1"
	"github.com/yndd/ndd-runtime/pkg/resource"
	"github.com/yndd/nddo-runtime/pkg/odr"
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
	GetOrganizationName() string
	GetDeploymentName() string
	GetIpamName() string
	GetNetworkInstanceName() string
	GetIpPrefix() string
	//GetPrefixLength() *uint32
	GetAddressFamily() string
	GetSourceTag() map[string]string
	GetSelector() map[string]string
	SetIpPrefix(p string)
	HasIpPrefix() (string, bool)
	SetOrganizationName(string)
	SetDeploymentName(string)
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

func (x *Register) GetOrganizationName() string {
	return odr.GetOrganizationName(x.GetNamespace())
}

func (x *Register) GetDeploymentName() string {
	return odr.GetDeploymentName(x.GetNamespace())
}

func (x *Register) GetIpamName() string {
	split := strings.Split(x.GetName(), ".")
	if len(split) > 2 {
		return split[0]
	}
	return ""
}

func (x *Register) GetNetworkInstanceName() string {
	split := strings.Split(x.GetName(), ".")
	if len(split) > 2 {
		return split[1]
	}
	return ""
}

func (n *Register) GetIpPrefix() string {
	if reflect.ValueOf(n.Spec.Register.IpPrefix).IsZero() {
		return ""
	}
	return *n.Spec.Register.IpPrefix
}

/*
func (n *Register) GetPrefixLength() *uint32 {
	if reflect.ValueOf(n.Spec.Register.PrefixLength).IsZero() {
		return nil
	}
	return n.Spec.Register.PrefixLength
}
*/

func (n *Register) GetAddressFamily() string {
	if reflect.ValueOf(n.Spec.Register.AddressFamily).IsZero() {
		return "ipv4"
	}
	return *n.Spec.Register.AddressFamily
}

func (n *Register) GetSourceTag() map[string]string {
	s := make(map[string]string)
	if reflect.ValueOf(n.Spec.Register.SourceTag).IsZero() {
		return s
	}
	for _, tag := range n.Spec.Register.SourceTag {
		s[*tag.Key] = *tag.Value
	}
	return s
}

func (n *Register) GetSelector() map[string]string {
	s := make(map[string]string)
	if reflect.ValueOf(n.Spec.Register.Selector).IsZero() {
		return s
	}
	for _, tag := range n.Spec.Register.Selector {
		s[*tag.Key] = *tag.Value
	}
	return s
}

func (n *Register) SetIpPrefix(p string) {
	n.Status = RegisterStatus{
		Register: &NddrIpamRegister{
			State: &NddrRegisterState{
				IpPrefix: &p,
			},
		},
	}
}

func (n *Register) HasIpPrefix() (string, bool) {
	if n.Status.Register != nil && n.Status.Register.State != nil && n.Status.Register.State.IpPrefix != nil {
		return *n.Status.Register.State.IpPrefix, true
	}
	return "", false

}

func (n *Register) SetOrganizationName(s string) {
	n.Status.OrganizationName = &s
}

func (n *Register) SetDeploymentName(s string) {
	n.Status.DeploymentName = &s
}

func (n *Register) SetIpamName(s string) {
	n.Status.IpamName = &s
}

func (n *Register) SetNetworkInstanceName(s string) {
	n.Status.NetworkInstanceName = &s
}
