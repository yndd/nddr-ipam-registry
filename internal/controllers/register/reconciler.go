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

package register

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/yndd/ndd-runtime/pkg/event"
	"github.com/yndd/ndd-runtime/pkg/logging"
	"github.com/yndd/nddo-runtime/pkg/odns"
	"github.com/yndd/nddo-runtime/pkg/reconciler/managed"
	"github.com/yndd/nddo-runtime/pkg/resource"
	ipamv1alpha1 "github.com/yndd/nddr-ipam-registry/apis/ipam/v1alpha1"
	"github.com/yndd/nddr-ipam-registry/internal/handler"
	"github.com/yndd/nddr-ipam-registry/internal/shared"
	"github.com/yndd/nddr-org-registry/pkg/registry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

const (
	// timers
	reconcileTimeout = 1 * time.Minute
	veryShortWait    = 1 * time.Second
	// errors
	errUnexpectedResource = "unexpected infrastructure object"
	errGetK8sResource     = "cannot get infrastructure resource"
)

// Setup adds a controller that reconciles infra.
func Setup(mgr ctrl.Manager, o controller.Options, nddcopts *shared.NddControllerOptions) error {
	name := "nddo/" + strings.ToLower(ipamv1alpha1.RegisterGroupKind)
	ipfn := func() ipamv1alpha1.Ip { return &ipamv1alpha1.Ipam{} }
	//rglfn := func() ipamv1alpha1.RgList { return &ipamv1alpha1.RegistryList{} }
	//rrfn := func() ipamv1alpha1.Rr { return &ipamv1alpha1.Register{} }
	//rrlfn := func() ipamv1alpha1.RrList { return &ipamv1alpha1.RegisterList{} }

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(ipamv1alpha1.RegisterGroupVersionKind),
		managed.WithLogger(nddcopts.Logger.WithValues("controller", name)),
		managed.WithApplication(&application{
			client: resource.ClientApplicator{
				Client:     mgr.GetClient(),
				Applicator: resource.NewAPIPatchingApplicator(mgr.GetClient()),
			},
			log:      nddcopts.Logger.WithValues("applogic", name),
			newIpam:  ipfn,
			handler:  nddcopts.Handler,
			registry: nddcopts.Registry,
		}),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&ipamv1alpha1.Register{}).
		Owns(&ipamv1alpha1.Register{}).
		WithEventFilter(resource.IgnoreUpdateWithoutGenerationChangePredicate()).
		WithEventFilter(resource.IgnoreUpdateWithoutGenerationChangePredicate()).
		Complete(r)

}

type application struct {
	client resource.ClientApplicator
	log    logging.Logger

	newIpam func() ipamv1alpha1.Ip

	//pool    map[string]hash.HashTable
	handler handler.Handler
	//speedy   map[string]int
	registry registry.Registry

	//poolmutex sync.Mutex
	//speedyMutex sync.Mutex
}

func getCrName(cr ipamv1alpha1.Rr) string {
	return strings.Join([]string{cr.GetNamespace(), cr.GetIpamName(), cr.GetNetworkInstanceName()}, ".")
}

func (r *application) Initialize(ctx context.Context, mg resource.Managed) error {
	return nil
}

func (r *application) Update(ctx context.Context, mg resource.Managed) (map[string]string, error) {
	cr, ok := mg.(*ipamv1alpha1.Register)
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
	/*
		cr, _ := mg.(*ipamv1alpha1.Registry)
		crName := getCrName(cr)
		r.speedyMutex.Lock()
		speedy := r.speedy[crName]
		r.speedyMutex.Unlock()
		if speedy <= 5 {
			r.log.Debug("Speedy", "number", speedy)
			speedy++
			return veryShortWait
		}
	*/
	return reconcileTimeout
}

func (r *application) Delete(ctx context.Context, mg resource.Managed) (bool, error) {
	cr, ok := mg.(*ipamv1alpha1.Register)
	if !ok {
		return true, errors.New(errUnexpectedResource)
	}
	log := r.log.WithValues("function", "handleAppLogic", "crname", cr.GetName())
	log.Debug("handleDelete")

	if prefix, ok := cr.HasIpPrefix(); ok {
		registerInfo := &handler.RegisterInfo{
			Namespace:           cr.GetNamespace(),
			RegistryName:        cr.GetIpamName(),
			Name:                cr.GetName(),
			NetworkInstanceName: odns.GetParentResourceName(cr.GetName()),
			CrName:              getCrName(cr),
			IpPrefix:            prefix,
			Selector:            cr.GetSelector(),
			SourceTag:           cr.GetSourceTag(),
		}

		log.Debug("resource dealloc", "registerInfo", registerInfo)

		if err := r.handler.DeRegister(ctx, registerInfo); err != nil {
			return true, err
		}
	}

	return true, nil
}

func (r *application) FinalDelete(ctx context.Context, mg resource.Managed) {

}

func (r *application) handleAppLogic(ctx context.Context, cr ipamv1alpha1.Rr) (map[string]string, error) {
	log := r.log.WithValues("function", "handleAppLogic", "crname", cr.GetName())
	log.Debug("handleAppLogic")

	selector := cr.GetSelector()
	if _, ok := selector[ipamv1alpha1.KeyPurpose]; !ok {
		return nil, errors.New("pupose not provided in resource request")
	}

	if _, ok := selector[ipamv1alpha1.KeyAddressFamily]; !ok {
		return nil, errors.New("af not provided in resource request")
	}

	registerInfo := &handler.RegisterInfo{
		Namespace:           cr.GetNamespace(),
		RegistryName:        cr.GetIpamName(),
		NetworkInstanceName: odns.GetParentResourceName(cr.GetName()),
		Name:                cr.GetName(),
		CrName:              getCrName(cr),
		Purpose:             selector[ipamv1alpha1.KeyPurpose],
		AddressFamily:       selector[ipamv1alpha1.KeyAddressFamily],
		IpPrefix:            cr.GetIpPrefix(),
		Selector:            selector,
		SourceTag:           cr.GetSourceTag(),
	}

	log.Debug("resource alloc", "registerInfo", registerInfo)

	ipPrefix, err := r.handler.Register(ctx, registerInfo)
	if err != nil {
		return nil, err
	}

	cr.SetIpPrefix(*ipPrefix)

	cr.SetOrganization(cr.GetOrganization())
	cr.SetDeployment(cr.GetDeployment())
	cr.SetAvailabilityZone(cr.GetAvailabilityZone())
	cr.SetIpamName(cr.GetIpamName())
	cr.SetNetworkInstanceName(cr.GetNetworkInstanceName())

	return nil, nil
}
