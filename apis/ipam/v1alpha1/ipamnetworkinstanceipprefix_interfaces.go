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
	"github.com/yndd/ndd-runtime/pkg/utils"
	nddov1 "github.com/yndd/nddo-runtime/apis/common/v1"
	"github.com/yndd/nddo-runtime/pkg/odns"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ IppList = &IpamNetworkInstanceIpPrefixList{}

// +k8s:deepcopy-gen=false
type IppList interface {
	client.ObjectList

	GetIpPrefixes() []Ipp
}

func (x *IpamNetworkInstanceIpPrefixList) GetIpPrefixes() []Ipp {
	xs := make([]Ipp, len(x.Items))
	for i, r := range x.Items {
		r := r // Pin range variable so we can take its address.
		xs[i] = &r
	}
	return xs
}

var _ Ipp = &IpamNetworkInstanceIpPrefix{}

// +k8s:deepcopy-gen=false
type Ipp interface {
	resource.Object
	resource.Conditioned

	GetCondition(ct nddv1.ConditionKind) nddv1.Condition
	SetConditions(c ...nddv1.Condition)
	GetOrganization() string
	GetDeployment() string
	GetAvailabilityZone() string
	GetIpamName() string
	GetNetworkInstanceName() string
	GetIpPrefixName() string
	GetIpPrefix() string
	GetPool() bool
	GetAdminState() string
	GetDescription() string
	GetTags() map[string]string
	InitializeResource() error
	SetStatus(string)
	SetReason(string)
	GetStatus() string
	GetAllocatedPrefixes() uint32

	SetOrganization(string)
	SetDeployment(string)
	SetAvailabilityZone(s string)
	SetIpamName(string)
	SetNetworkInstanceName(string)
	SetIpPrefixName(string)
	SetAddressFamily(string)
}

// GetCondition of this Network Node.
func (x *IpamNetworkInstanceIpPrefix) GetCondition(ct nddv1.ConditionKind) nddv1.Condition {
	return x.Status.GetCondition(ct)
}

// SetConditions of the Network Node.
func (x *IpamNetworkInstanceIpPrefix) SetConditions(c ...nddv1.Condition) {
	x.Status.SetConditions(c...)
}

func (x *IpamNetworkInstanceIpPrefix) GetOrganization() string {
	return odns.Name2OdnsRegistryNi(x.GetName()).GetOrganization()
}

func (x *IpamNetworkInstanceIpPrefix) GetDeployment() string {
	return odns.Name2OdnsRegistryNi(x.GetName()).GetDeployment()
}

func (x *IpamNetworkInstanceIpPrefix) GetAvailabilityZone() string {
	return odns.Name2OdnsRegistryNi(x.GetName()).GetAvailabilityZone()
}

func (x *IpamNetworkInstanceIpPrefix) GetIpamName() string {
	return odns.Name2OdnsRegistryNi(x.GetName()).GetRegistryName()
}

func (x *IpamNetworkInstanceIpPrefix) GetNetworkInstanceName() string {
	return odns.Name2OdnsRegistryNi(x.GetName()).GetNetworkInstanceName()
}

func (x *IpamNetworkInstanceIpPrefix) GetIpPrefixName() string {
	return odns.Name2OdnsRegistryNi(x.GetName()).GetResourceName()
}

func (x *IpamNetworkInstanceIpPrefix) GetIpPrefix() string {
	if reflect.ValueOf(x.Spec.IpamNetworkInstanceIpPrefix.Prefix).IsZero() {
		return ""
	}
	return *x.Spec.IpamNetworkInstanceIpPrefix.Prefix
}

func (x *IpamNetworkInstanceIpPrefix) GetPool() bool {
	if reflect.ValueOf(x.Spec.IpamNetworkInstanceIpPrefix.Pool).IsZero() {
		return false
	}
	return *x.Spec.IpamNetworkInstanceIpPrefix.Pool
}

func (x *IpamNetworkInstanceIpPrefix) GetAdminState() string {
	if reflect.ValueOf(x.Spec.IpamNetworkInstanceIpPrefix.AdminState).IsZero() {
		return ""
	}
	return *x.Spec.IpamNetworkInstanceIpPrefix.AdminState
}

func (x *IpamNetworkInstanceIpPrefix) GetDescription() string {
	if reflect.ValueOf(x.Spec.IpamNetworkInstanceIpPrefix.Description).IsZero() {
		return ""
	}
	return *x.Spec.IpamNetworkInstanceIpPrefix.Description
}

func (x *IpamNetworkInstanceIpPrefix) GetTags() map[string]string {
	s := make(map[string]string)
	if reflect.ValueOf(x.Spec.IpamNetworkInstanceIpPrefix.Tag).IsZero() {
		return s
	}
	for _, tag := range x.Spec.IpamNetworkInstanceIpPrefix.Tag {
		s[*tag.Key] = *tag.Value
	}
	return s
}

func (x *IpamNetworkInstanceIpPrefix) InitializeResource() error {
	tags := make([]*nddov1.Tag, 0, len(x.Spec.IpamNetworkInstanceIpPrefix.Tag))
	for _, tag := range x.Spec.IpamNetworkInstanceIpPrefix.Tag {
		tags = append(tags, &nddov1.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}

	if x.Status.IpamNetworkInstanceIpPrefix != nil {
		// pool was already initialiazed
		// copy the spec, but not the state
		x.Status.IpamNetworkInstanceIpPrefix.AdminState = x.Spec.IpamNetworkInstanceIpPrefix.AdminState
		x.Status.IpamNetworkInstanceIpPrefix.Description = x.Spec.IpamNetworkInstanceIpPrefix.Description
		x.Status.IpamNetworkInstanceIpPrefix.Prefix = x.Spec.IpamNetworkInstanceIpPrefix.Prefix
		x.Status.IpamNetworkInstanceIpPrefix.Pool = x.Spec.IpamNetworkInstanceIpPrefix.Pool
		x.Status.IpamNetworkInstanceIpPrefix.Tag = tags
		return nil
	}

	x.Status.IpamNetworkInstanceIpPrefix = &NddrIpamIpamNetworkInstanceIpPrefix{
		AdminState:  x.Spec.IpamNetworkInstanceIpPrefix.AdminState,
		Description: x.Spec.IpamNetworkInstanceIpPrefix.Description,
		Prefix:      x.Spec.IpamNetworkInstanceIpPrefix.Prefix,
		Pool:        x.Spec.IpamNetworkInstanceIpPrefix.Pool,
		Tag:         tags,
		State: &NddrIpamIpamNetworkInstanceIpPrefixState{
			Status:   utils.StringPtr(""),
			Reason:   utils.StringPtr(""),
			Tag:      make([]*nddov1.Tag, 0),
			Adresses: utils.Uint32Ptr(0),
			Child:    &NddrIpamIpamNetworkInstanceIpPrefixStateChild{},
			Parent:   &NddrIpamIpamNetworkInstanceIpPrefixStateParent{},
		},
	}
	return nil
}

func (x *IpamNetworkInstanceIpPrefix) SetStatus(s string) {
	x.Status.IpamNetworkInstanceIpPrefix.State.Status = &s
}

func (x *IpamNetworkInstanceIpPrefix) SetReason(s string) {
	x.Status.IpamNetworkInstanceIpPrefix.State.Reason = &s
}

func (x *IpamNetworkInstanceIpPrefix) GetStatus() string {
	if x.Status.IpamNetworkInstanceIpPrefix != nil && x.Status.IpamNetworkInstanceIpPrefix.State != nil && x.Status.IpamNetworkInstanceIpPrefix.State.Status != nil {
		return *x.Status.IpamNetworkInstanceIpPrefix.State.Status
	}
	return "unknown"
}

func (x *IpamNetworkInstanceIpPrefix) GetAllocatedPrefixes() uint32 {
	if x.Status.IpamNetworkInstanceIpPrefix != nil && x.Status.IpamNetworkInstanceIpPrefix.State != nil && x.Status.IpamNetworkInstanceIpPrefix.State.Adresses != nil {
		return *x.Status.IpamNetworkInstanceIpPrefix.State.Adresses
	}
	return 0
}

func (x *IpamNetworkInstanceIpPrefix) SetOrganization(s string) {
	x.Status.SetOrganization(s)
}

func (x *IpamNetworkInstanceIpPrefix) SetDeployment(s string) {
	x.Status.SetDeployment(s)
}

func (x *IpamNetworkInstanceIpPrefix) SetAvailabilityZone(s string) {
	x.Status.SetAvailabilityZone(s)
}

func (x *IpamNetworkInstanceIpPrefix) SetIpamName(s string) {
	x.Status.RegistryName = &s
}

func (x *IpamNetworkInstanceIpPrefix) SetNetworkInstanceName(s string) {
	x.Status.NetworkInstanceName = &s
}

func (x *IpamNetworkInstanceIpPrefix) SetIpPrefixName(s string) {
	x.Status.IpPrefixName = &s
}

func (x *IpamNetworkInstanceIpPrefix) SetAddressFamily(s string) {
	for _, tag := range x.Status.IpamNetworkInstanceIpPrefix.State.Tag {
		if *tag.Key == KeyAddressFamily {
			tag.Value = &s
			return
		}
	}
	x.Status.IpamNetworkInstanceIpPrefix.State.Tag = append(x.Status.IpamNetworkInstanceIpPrefix.State.Tag, &nddov1.Tag{
		Key:   utils.StringPtr(KeyAddressFamily),
		Value: &s,
	})
}
