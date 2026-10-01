// Package payments abstracts payment providers (Mobile Money, cards,
// internal wallet). The MVP ships a simulated provider; real providers
// (Orange Money, MTN MoMo, Wave, card acquirers) implement Provider.
package payments

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// HoldRequest asks the provider to collect and hold (escrow) a payment.
type HoldRequest struct {
	JobID    string
	PayerID  string
	Amount   int // FCFA
	Currency string
}

// Provider is implemented by every payment integration.
type Provider interface {
	Name() string
	// Hold collects the amount from the payer and keeps it in escrow.
	Hold(ctx context.Context, req HoldRequest) (ref string, err error)
	// Release settles held funds (the worker's share goes to the wallet).
	Release(ctx context.Context, ref string, amount int) error
	// Refund returns amount of the held funds to the payer.
	Refund(ctx context.Context, ref string, amount int) error
}

// Simulated accepts every operation; used for the prototype (§38).
type Simulated struct{}

func (Simulated) Name() string { return "simulated" }

func (Simulated) Hold(_ context.Context, req HoldRequest) (string, error) {
	if req.Amount < 0 {
		return "", fmt.Errorf("invalid amount %d", req.Amount)
	}
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "sim_" + hex.EncodeToString(b), nil
}

func (Simulated) Release(context.Context, string, int) error { return nil }
func (Simulated) Refund(context.Context, string, int) error  { return nil }

// New returns the provider configured by name.
func New(name string) (Provider, error) {
	switch name {
	case "", "simulated":
		return Simulated{}, nil
	}
	return nil, fmt.Errorf("unknown payment provider %q", name)
}
