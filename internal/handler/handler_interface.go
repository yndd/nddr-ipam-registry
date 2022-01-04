package handler

import (
	"context"

	"github.com/yndd/ndd-runtime/pkg/logging"
	ipamv1alpha1 "github.com/yndd/nddr-ipam-registry/apis/ipam/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Option can be used to manipulate Options.
type Option func(Handler)

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(log logging.Logger) Option {
	return func(s Handler) {
		s.WithLogger(log)
	}
}

func WithClient(c client.Client) Option {
	return func(s Handler) {
		s.WithClient(c)
	}
}

type Handler interface {
	WithLogger(log logging.Logger)
	WithClient(a client.Client)
	Init(string)
	Delete(string)
	CheckAllocation(crName string, cr ipamv1alpha1.Rr) (bool, error)
	ResetSpeedy(string)
	GetSpeedy(crName string) int
	IncrementSpeedy(crName string)
	Register(context.Context, *RegisterInfo) (*string, error)
	DeRegister(context.Context, *RegisterInfo) error
	AddIpPrefix(crName string, cr ipamv1alpha1.Ipp) error
}
