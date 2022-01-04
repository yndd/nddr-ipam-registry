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

package grpcserver

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/yndd/nddo-grpc/resource/resourcepb"
	ipamv1alpha1 "github.com/yndd/nddr-ipam-registry/apis/ipam/v1alpha1"
	"github.com/yndd/nddr-ipam-registry/internal/handler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func (r *server) ResourceGet(ctx context.Context, req *resourcepb.Request) (*resourcepb.Reply, error) {
	log := r.log.WithValues("Request", req)
	log.Debug("ResourceGet...")

	return &resourcepb.Reply{Ready: true}, nil
}

func (r *server) ResourceRequest(ctx context.Context, req *resourcepb.Request) (*resourcepb.Reply, error) {
	log := r.log.WithValues("Request", req)

	namespace := req.GetNamespace()
	registryName := strings.Split(req.GetResourceName(), ".")[0]

	if _, ok := req.GetRequest().GetSelector()[ipamv1alpha1.KeyPurpose]; !ok {
		return nil, errors.New("pupose not provided in resource request")
	}

	if _, ok := req.GetRequest().GetSelector()[ipamv1alpha1.KeyAddressFamily]; !ok {
		return nil, errors.New("af not provided in resource request")
	}

	registerInfo := &handler.RegisterInfo{
		Namespace:     namespace,
		RegistryName:  registryName,
		RegisterName:  req.GetResourceName(),
		CrName:        strings.Join([]string{namespace, registryName}, "."),
		IpPrefix:      req.GetRequest().GetIpPrefix(),
		Purpose:       req.GetRequest().GetSelector()[ipamv1alpha1.KeyPurpose],
		AddressFamily: req.GetRequest().GetSelector()[ipamv1alpha1.KeyAddressFamily],
		Selector:      req.GetRequest().GetSelector(),
		SourceTag:     req.GetRequest().GetSourceTag(),
	}

	log.Debug("resource alloc", "registerInfo", registerInfo)

	prefix, err := r.handler.Register(ctx, registerInfo)
	if err != nil {
		return &resourcepb.Reply{Ready: false}, err
	}

	// send a generic event to trigger a registry reconciliation based on a new allocation
	// to update the status
	r.eventChs[ipamv1alpha1.IpamGroupKind] <- event.GenericEvent{
		Object: &ipamv1alpha1.Register{
			ObjectMeta: metav1.ObjectMeta{Name: req.GetResourceName(), Namespace: namespace},
		},
	}

	return &resourcepb.Reply{
		Ready:      true,
		Timestamp:  time.Now().UnixNano(),
		ExpiryTime: time.Now().UnixNano(),
		Data: map[string]*resourcepb.TypedValue{
			"ip-prefix": {Value: &resourcepb.TypedValue_StringVal{StringVal: *prefix}},
		},
	}, nil
}

func (r *server) ResourceRelease(ctx context.Context, req *resourcepb.Request) (*resourcepb.Reply, error) {
	log := r.log.WithValues("Request", req)
	log.Debug("ResourceDeAlloc...")

	namespace := req.GetNamespace()
	registryName := strings.Split(req.GetResourceName(), ".")[0]

	registerInfo := &handler.RegisterInfo{
		Namespace:    namespace,
		RegistryName: registryName,
		CrName:       strings.Join([]string{namespace, registryName}, "."),
		Selector:     req.Request.Selector,
		SourceTag:    req.Request.SourceTag,
	}

	log.Debug("resource dealloc", "registerInfo", registerInfo)

	if err := r.handler.DeRegister(ctx, registerInfo); err != nil {
		return &resourcepb.Reply{Ready: false}, err
	}

	// send a generic event to trigger a registry reconciliation based on a new DeAllocation
	r.eventChs[ipamv1alpha1.IpamGroupKind] <- event.GenericEvent{
		Object: &ipamv1alpha1.Register{
			ObjectMeta: metav1.ObjectMeta{Name: req.GetResourceName(), Namespace: namespace},
		},
	}

	return &resourcepb.Reply{Ready: true}, nil
}
