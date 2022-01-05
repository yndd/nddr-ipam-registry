package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hansthienpondt/goipam/pkg/table"
	"github.com/pkg/errors"
	"github.com/yndd/ndd-runtime/pkg/logging"
	ipamv1alpha1 "github.com/yndd/nddr-ipam-registry/apis/ipam/v1alpha1"
	"inet.af/netaddr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(opts ...Option) (Handler, error) {
	ipamNifn := func() ipamv1alpha1.In { return &ipamv1alpha1.IpamNetworkInstance{} }
	s := &handler{
		iptree:                 make(map[string]*table.RouteTable),
		speedy:                 make(map[string]int),
		newIpamNetworkInstance: ipamNifn,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

func (r *handler) WithLogger(log logging.Logger) {
	r.log = log
}

func (r *handler) WithClient(c client.Client) {
	r.client = c
}

type RegisterInfo struct {
	Namespace           string
	Name                string
	RegistryName        string
	NetworkInstanceName string // only used in ipam
	CrName              string
	IpPrefix            string
	Purpose             string
	AddressFamily       string
	Selector            map[string]string
	SourceTag           map[string]string
}

type handler struct {
	log logging.Logger
	// kubernetes
	client client.Client

	newIpamNetworkInstance func() ipamv1alpha1.In
	iptreeMutex            sync.Mutex
	iptree                 map[string]*table.RouteTable
	speedyMutex            sync.Mutex
	speedy                 map[string]int
}

func (r *handler) Init(crName string) {
	r.iptreeMutex.Lock()
	defer r.iptreeMutex.Unlock()
	if _, ok := r.iptree[crName]; !ok {
		r.iptree[crName] = table.NewRouteTable()
	}

	r.speedyMutex.Lock()
	defer r.speedyMutex.Unlock()
	if _, ok := r.speedy[crName]; !ok {
		r.speedy[crName] = 0
	}
}

func (r *handler) Delete(crName string) {
	r.iptreeMutex.Lock()
	defer r.iptreeMutex.Unlock()
	delete(r.iptree, crName)

	r.speedyMutex.Lock()
	defer r.speedyMutex.Unlock()
	delete(r.speedy, crName)
}

func (r *handler) CheckAllocation(crName string, cr ipamv1alpha1.Rr) (bool, error) {
	// check if allocations exists
	if _, ok := r.iptree[crName]; ok {
		if prefix, ok := cr.HasIpPrefix(); ok {
			p, err := netaddr.ParseIPPrefix(prefix)
			if err != nil {
				return false, err
			}
			/*
				routes := r.iptree[treename].Children(p)
			*/
			/*
				if len(routes) > 0 {
					// We cannot delete the prefix yet due to existing allocations
					record.Event(cr, event.Warning(reasonCannotDeleteDueToAllocations, err))
					log.Debug("Cannot delete prefix due to existing allocations", "error", err)
					return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
				}
			*/

			route := table.NewRoute(p)
			// TODO do we need to add the labels or not
			//route.UpdateLabel(cr.GetTags())

			if _, _, err := r.iptree[crName].Delete(route); err != nil {
				return false, err
			}
		}
	}
	return false, nil
}

func (r *handler) ResetSpeedy(crName string) {
	r.speedyMutex.Lock()
	defer r.speedyMutex.Unlock()
	if _, ok := r.speedy[crName]; ok {
		r.speedy[crName] = 0
	}
}

func (r *handler) GetSpeedy(crName string) int {
	r.speedyMutex.Lock()
	defer r.speedyMutex.Unlock()
	if _, ok := r.speedy[crName]; ok {
		return r.speedy[crName]
	}
	return 9999
}

func (r *handler) IncrementSpeedy(crName string) {
	r.speedyMutex.Lock()
	defer r.speedyMutex.Unlock()
	if _, ok := r.speedy[crName]; ok {
		r.speedy[crName]++
	}
}

func (r *handler) Register(ctx context.Context, info *RegisterInfo) (*string, error) {
	ni, iptree, err := r.validateRegister(ctx, info)
	if err != nil {
		return nil, err
	}

	// the selector is used in the tree to find the entry in the tree
	// we use all the keys in the source-tag and selector for the search
	fullselector := labels.NewSelector()
	l := make(map[string]string)
	for key, val := range info.Selector {
		req, err := labels.NewRequirement(key, selection.In, []string{val})
		if err != nil {
			r.log.Debug("wrong object", "Error", err)
			return nil, err
		}
		fullselector = fullselector.Add(*req)
		l[key] = val
	}
	for key, val := range info.SourceTag {
		req, err := labels.NewRequirement(key, selection.In, []string{val})
		if err != nil {
			r.log.Debug("wrong object", "Error", err)
			return nil, err
		}
		fullselector = fullselector.Add(*req)
		l[key] = val
	}
	prefix := info.IpPrefix
	if prefix != "" {
		r.log.Debug("alloc has prefix", "prefix", prefix)

		// via selector perform allocation
		selector := labels.NewSelector()
		for key, val := range info.Selector {
			req, err := labels.NewRequirement(key, selection.In, []string{val})
			if err != nil {
				r.log.Debug("wrong object", "Error", err)
				return nil, err
			}
			selector = selector.Add(*req)
		}

		a, err := netaddr.ParseIPPrefix(prefix)
		if err != nil {
			r.log.Debug("Cannot parse ip prefix", "error", err)
			return nil, errors.Wrap(err, "Cannot parse ip prefix")
		}
		route := table.NewRoute(a)
		route.UpdateLabel(l)

		if err := iptree.Add(route); err != nil {
			r.log.Debug("route insertion failed")
			if !strings.Contains(err.Error(), "already exists") {
				return nil, errors.Wrap(err, "route insertion failed")
			}
		}
		prefix = route.String()
	} else {
		// alloc has no prefix assigned, try to assign prefix
		// check if the prefix already exists
		routes := iptree.GetByLabel(fullselector)
		if len(routes) == 0 {
			// allocate prefix
			r.log.Debug("Query not found, allocate a prefix")

			// via selector perform allocation
			selector := labels.NewSelector()
			for key, val := range info.Selector {
				req, err := labels.NewRequirement(key, selection.In, []string{val})
				if err != nil {
					r.log.Debug("wrong object", "Error", err)
					return nil, errors.Wrap(err, "wrong object")
				}
				selector = selector.Add(*req)
			}

			routes := iptree.GetByLabel(selector)

			// required during startup when not everything is initialized
			// we break and the reconciliation will take care
			if len(routes) == 0 {
				r.log.Debug("no available routes")
				return nil, errors.New("no available routes")
			}

			// TBD we take the first prefix
			prefixLength, err := getPrefixLength(info, ni)
			if err != nil {
				return nil, errors.Wrap(err, "prefix Length not properly configured")
			}

			a, ok := iptree.FindFreePrefix(routes[0].IPPrefix(), uint8(prefixLength))
			if !ok {
				r.log.Debug("allocation failed")
				return nil, errors.New("allocation failed")
			}

			route := table.NewRoute(a)
			route.UpdateLabel(l)
			if err := iptree.Add(route); err != nil {
				r.log.Debug("route insertion failed")
				return nil, errors.Wrap(err, "route insertion failed")
			}
			prefix = route.String()

		} else {
			if len(routes) > 1 {
				// this should never happen since the labels should provide uniqueness
				r.log.Debug("strange situation, route in tree found multiple times", "ases", routes)
			}
			prefix = routes[0].IPPrefix().String()
		}

	}
	return &prefix, nil
}

func (r *handler) DeRegister(ctx context.Context, info *RegisterInfo) error {

	_, iptree, err := r.validateRegister(ctx, info)
	if err != nil {
		return err
	}

	p, err := netaddr.ParseIPPrefix(info.IpPrefix)
	if err != nil {
		return err
	}
	/*
		routes := r.iptree[treename].Children(p)
	*/
	/*
		if len(routes) > 0 {
			// We cannot delete the prefix yet due to existing allocations
			record.Event(cr, event.Warning(reasonCannotDeleteDueToAllocations, err))
			log.Debug("Cannot delete prefix due to existing allocations", "error", err)
			return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
		}
	*/

	route := table.NewRoute(p)
	// TODO do we need to add the labels or not
	//route.UpdateLabel(cr.GetTags())

	if _, _, err := iptree.Delete(route); err != nil {
		r.log.Debug("IPPrefix deleteion failed", "prefix", p)
		return err
	}
	return nil

}

func (r *handler) validateRegister(ctx context.Context, info *RegisterInfo) (ipamv1alpha1.In, *table.RouteTable, error) {
	namespace := info.Namespace
	//registryName := info.RegistryName
	crName := info.CrName
	networkInstanceName := info.NetworkInstanceName

	// find registry in k8s api
	ni := r.newIpamNetworkInstance()
	if err := r.client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      networkInstanceName}, ni); err != nil {
		// can happen when the ipam is not found
		r.log.Debug("networkInstance not found")
		return nil, nil, errors.Wrap(err, "networkInstance not found")
	}

	// check is registry is ready
	if ni.GetCondition(ipamv1alpha1.ConditionKindReady).Status != corev1.ConditionTrue {
		return nil, nil, errors.New("networkInstance not ready")
	}

	if _, ok := r.iptree[crName]; !ok {
		return nil, nil, errors.New("networkInstance iptree not ready")
	}

	// check if the pool/register is ready to handle new registrations
	r.iptreeMutex.Lock()
	defer r.iptreeMutex.Unlock()
	if _, ok := r.iptree[crName]; !ok {
		r.log.Debug("pool/tree not ready", "crName", crName)
		return nil, nil, fmt.Errorf("pool/tree not ready, crName: %s", crName)
	}
	iptree := r.iptree[crName]

	return ni, iptree, nil
}

func getPrefixLength(info *RegisterInfo, ni ipamv1alpha1.In) (uint32, error) {
	prefixLength := ni.GetDefaultPrefixLength(info.Purpose, info.AddressFamily)
	if prefixLength == nil {
		return 0, fmt.Errorf("default prefix length not configured properly, purpose: %s, sf: %s", info.Purpose, info.AddressFamily)
	}
	return *prefixLength, nil
}

func (r *handler) AddIpPrefix(crName string, cr ipamv1alpha1.Ipp) error {
	r.speedyMutex.Lock()
	defer r.speedyMutex.Unlock()
	if _, ok := r.iptree[crName]; !ok {
		r.log.Debug("Parent Routing table not ready")
		return errors.New("ipam ni not ready")
	}

	p, err := netaddr.ParseIPPrefix(cr.GetIpPrefix())
	if err != nil {
		r.log.Debug("UpdateConfig ParseIPPrefix", "Error", err)
		return errors.Wrap(err, "ParseIPPrefix failed")
	}
	// we derive the address family from the prefix, to avoid exposing it to the user
	var af string
	if p.IP().Is4() {
		af = string(ipamv1alpha1.AddressFamilyIpv4)
	}
	if p.IP().Is6() {
		af = string(ipamv1alpha1.AddressFamilyIpv6)
	}
	cr.SetAddressFamily(af)
	// we add the address family in the tag/label to allow to selec the prefix on this basis
	tags := cr.GetTags()
	tags[ipamv1alpha1.KeyAddressFamily] = af
	route := table.NewRoute(p)
	route.UpdateLabel(tags)

	if err := r.iptree[crName].Add(route); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		r.log.Debug("IPPrefix insertion failed", "prefix", p)
		return errors.Wrap(err, "IPPrefix insertion failed")
	}
	return nil
}
