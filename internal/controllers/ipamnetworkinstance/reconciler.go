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

package ipamnetworkinstance

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/yndd/ndd-runtime/pkg/event"
	"github.com/yndd/ndd-runtime/pkg/logging"
	"github.com/yndd/nddo-runtime/pkg/reconciler/managed"
	"github.com/yndd/nddo-runtime/pkg/resource"
	ipamv1alpha1 "github.com/yndd/nddr-ipam-registry/apis/ipam/v1alpha1"
	"github.com/yndd/nddr-ipam-registry/internal/handler"
	"github.com/yndd/nddr-ipam-registry/internal/shared"
	"github.com/yndd/nddr-org-registry/pkg/registry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	gevent "sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// timers
	reconcileTimeout = 1 * time.Minute
	shortWait        = 5 * time.Second
	veryShortWait    = 1 * time.Second
	// errors
	errUnexpectedResource = "unexpected infrastructure object"
	errGetK8sResource     = "cannot get infrastructure resource"
)

// Setup adds a controller that reconciles infra.
func Setup(mgr ctrl.Manager, o controller.Options, nddcopts *shared.NddControllerOptions) (string, chan gevent.GenericEvent, error) {
	name := "nddo/" + strings.ToLower(ipamv1alpha1.IpamNetworkInstanceGroupKind)
	ipfn := func() ipamv1alpha1.Ip { return &ipamv1alpha1.Ipam{} }
	infn := func() ipamv1alpha1.In { return &ipamv1alpha1.IpamNetworkInstance{} }
	inlfn := func() ipamv1alpha1.InList { return &ipamv1alpha1.IpamNetworkInstanceList{} }
	//rrfn := func() ipamv1alpha1.Rr { return &ipamv1alpha1.Register{} }
	//rrlfn := func() ipamv1alpha1.RrList { return &ipamv1alpha1.RegisterList{} }

	events := make(chan gevent.GenericEvent)
	//speedy := make(map[string]int)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(ipamv1alpha1.IpamNetworkInstanceGroupVersionKind),
		managed.WithLogger(nddcopts.Logger.WithValues("controller", name)),
		managed.WithApplication(&application{
			client: resource.ClientApplicator{
				Client:     mgr.GetClient(),
				Applicator: resource.NewAPIPatchingApplicator(mgr.GetClient()),
			},
			log:                        nddcopts.Logger.WithValues("applogic", name),
			newIpam:                    ipfn,
			newIpamNetworkInstance:     infn,
			newIpamNetworkInstanceList: inlfn,
			registry:                   nddcopts.Registry,
			handler:                    nddcopts.Handler,
		}),
		//managed.WithSpeedy(speedy),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	)

	registerHandler := &EnqueueRequestForAllRegisters{
		client:                     mgr.GetClient(),
		log:                        nddcopts.Logger,
		ctx:                        context.Background(),
		handler:                    nddcopts.Handler,
		newIpamNetworkInstanceList: inlfn,
	}

	ipamHandler := &EnqueueRequestForAllIpams{
		client:                     mgr.GetClient(),
		log:                        nddcopts.Logger,
		ctx:                        context.Background(),
		handler:                    nddcopts.Handler,
		newIpamNetworkInstanceList: inlfn,
	}

	return ipamv1alpha1.IpamNetworkInstanceGroupKind, events, ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&ipamv1alpha1.IpamNetworkInstance{}).
		Owns(&ipamv1alpha1.IpamNetworkInstance{}).
		WithEventFilter(resource.IgnoreUpdateWithoutGenerationChangePredicate()).
		Watches(&source.Kind{Type: &ipamv1alpha1.Ipam{}}, ipamHandler).
		Watches(&source.Kind{Type: &ipamv1alpha1.Register{}}, registerHandler).
		Watches(&source.Channel{Source: events}, registerHandler).
		Complete(r)

}

type application struct {
	client resource.ClientApplicator
	log    logging.Logger

	newIpam                    func() ipamv1alpha1.Ip
	newIpamNetworkInstance     func() ipamv1alpha1.In
	newIpamNetworkInstanceList func() ipamv1alpha1.InList

	registry registry.Registry
	handler  handler.Handler
}

func getCrName(cr ipamv1alpha1.In) string {
	return strings.Join([]string{cr.GetNamespace(), cr.GetIpamName(), cr.GetNetworkInstanceName()}, ".")
}

func (r *application) Initialize(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*ipamv1alpha1.IpamNetworkInstance)
	if !ok {
		return errors.New(errUnexpectedResource)
	}

	if err := cr.InitializeResource(); err != nil {
		r.log.Debug("Cannot initialize", "error", err)
		return err
	}

	return nil
}

func (r *application) Update(ctx context.Context, mg resource.Managed) (map[string]string, error) {
	cr, ok := mg.(*ipamv1alpha1.IpamNetworkInstance)
	if !ok {
		return nil, errors.New(errUnexpectedResource)
	}

	return r.handleAppLogic(ctx, cr)
}

func (r *application) FinalUpdate(ctx context.Context, mg resource.Managed) {
	//cr, _ := mg.(*ipamv1alpha1.Registry)
	//crName := getCrName(cr)
	//r.infra[crName].PrintNodes(crName)
}

func (r *application) Timeout(ctx context.Context, mg resource.Managed) time.Duration {
	cr, _ := mg.(*ipamv1alpha1.IpamNetworkInstance)
	crName := getCrName(cr)
	speedy := r.handler.GetSpeedy(crName)
	if speedy <= 2 {
		r.handler.IncrementSpeedy(crName)
		r.log.Debug("Speedy incr", "number", r.handler.GetSpeedy(crName))
		switch speedy {
		case 0:
			return veryShortWait
		case 1, 2:
			return shortWait
		}

	}
	return reconcileTimeout
}

func (r *application) Delete(ctx context.Context, mg resource.Managed) (bool, error) {
	cr, ok := mg.(*ipamv1alpha1.IpamNetworkInstance)
	if !ok {
		return false, errors.New(errUnexpectedResource)
	}
	crName := getCrName(cr)

	r.log.Debug("delete", "crName", crName)

	// TODO check for allocations, etc etc
	return true, nil
}

func (r *application) FinalDelete(ctx context.Context, mg resource.Managed) {
	cr, ok := mg.(*ipamv1alpha1.IpamNetworkInstance)
	if !ok {
		return
	}
	crName := getCrName(cr)
	r.handler.Delete(crName)
}

func (r *application) handleAppLogic(ctx context.Context, cr ipamv1alpha1.In) (map[string]string, error) {
	log := r.log.WithValues("function", "handleAppLogic", "crname", cr.GetName())
	log.Debug("handleAppLogic")

	//organizationName := cr.GetOrganizationName()
	// get the deployment
	ipam := r.newIpam()
	if err := r.client.Get(ctx, types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      cr.GetIpamName()}, ipam); err != nil {
		// can happen when the deployment is not found
		cr.SetStatus("down")
		cr.SetReason("ipam not found")
		return nil, errors.Wrap(err, "ipam not found")
	}
	if ipam.GetCondition(ipamv1alpha1.ConditionKindReady).Status != corev1.ConditionTrue {
		cr.SetStatus("down")
		cr.SetReason("ipam not ready")
		return nil, errors.New("ipam not ready")
	}

	// initialize speedy
	crName := getCrName(cr)
	r.handler.Init(crName)
	// update status based on a scan of the pool

	cr.SetOrganizationName(cr.GetOrganizationName())
	cr.SetDeploymentName(cr.GetDeploymentName())
	cr.SetIpamName(cr.GetIpamName())

	// trick to use speedy for fast updates
	return map[string]string{"dummy": "dummy"}, nil
}
