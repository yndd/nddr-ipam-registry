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

package ipamnetworkinstanceipprefix

import (
	"context"

	//ndddvrv1 "github.com/yndd/ndd-core/apis/dvr/v1"

	"github.com/yndd/ndd-runtime/pkg/logging"
	ipamv1alpha1 "github.com/yndd/nddr-ipam-registry/apis/ipam/v1alpha1"
	"github.com/yndd/nddr-ipam-registry/internal/handler"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type EnqueueRequestForAllIpams struct {
	client client.Client
	log    logging.Logger
	ctx    context.Context

	handler handler.Handler

	newIpamNetworkInstanceIpPrefixList func() ipamv1alpha1.IppList
}

// Create enqueues a request for all infrastructures which pertains to the topology.
func (e *EnqueueRequestForAllIpams) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.add(evt.Object, q)
}

// Create enqueues a request for all infrastructures which pertains to the topology.
func (e *EnqueueRequestForAllIpams) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	e.add(evt.ObjectOld, q)
	e.add(evt.ObjectNew, q)
}

// Create enqueues a request for all infrastructures which pertains to the topology.
func (e *EnqueueRequestForAllIpams) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.add(evt.Object, q)
}

// Create enqueues a request for all infrastructures which pertains to the topology.
func (e *EnqueueRequestForAllIpams) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	e.add(evt.Object, q)
}

func (e *EnqueueRequestForAllIpams) add(obj runtime.Object, queue adder) {
	dd, ok := obj.(*ipamv1alpha1.Ipam)
	if !ok {
		return
	}
	log := e.log.WithValues("function", "watch ipam", "name", dd.GetName())
	log.Debug("ipamnetworkinstance handleEvent")

	d := e.newIpamNetworkInstanceIpPrefixList()
	if err := e.client.List(e.ctx, d); err != nil {
		return
	}

	for _, ipp := range d.GetIpPrefixes() {
		// only enqueue if the org and/or deployment name match
		if ipp.GetOrganization() == dd.GetOrganization() &&
			ipp.GetDeployment() == dd.GetDeployment() &&
			ipp.GetIpamName() == dd.GetIpamName() {
			crName := getCrName(ipp)
			e.handler.ResetSpeedy(crName)

			queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: ipp.GetNamespace(),
				Name:      ipp.GetName()}})
		}
	}
}
