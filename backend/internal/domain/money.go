package domain

import (
	"errors"
	"math"
)

// Split is the breakdown of a payment between worker and platform.
type Split struct {
	Amount       int
	PlatformFee  int
	WorkerAmount int
}

// ErrInvalidRate is returned for commission rates outside [0, 1].
var ErrInvalidRate = errors.New("commission rate must be between 0 and 1")

// SplitPayment computes the platform fee (rounded to the nearest FCFA) and
// what the worker receives. 10 000 at 10% → fee 1 000, worker 9 000.
func SplitPayment(amount int, rate float64) (Split, error) {
	if rate < 0 || rate > 1 || math.IsNaN(rate) {
		return Split{}, ErrInvalidRate
	}
	if amount < 0 {
		return Split{}, errors.New("amount must be positive")
	}
	fee := int(math.Round(float64(amount) * rate))
	return Split{Amount: amount, PlatformFee: fee, WorkerAmount: amount - fee}, nil
}

// PartialRefund splits held funds when a dispute is settled partially: the
// client gets refund back, the rest is paid out with the usual commission.
func PartialRefund(amount, refund int, rate float64) (Split, error) {
	if refund < 0 || refund > amount {
		return Split{}, errors.New("refund must be between 0 and the payment amount")
	}
	return SplitPayment(amount-refund, rate)
}
