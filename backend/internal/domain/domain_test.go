package domain

import (
	"math"
	"testing"
)

func TestSplitPayment(t *testing.T) {
	cases := []struct {
		amount      int
		rate        float64
		fee, worker int
	}{
		{10000, 0.10, 1000, 9000},
		{20000, 0.10, 2000, 18000},
		{15000, 0.125, 1875, 13125},
		{999, 0.10, 100, 899},
		{5000, 0, 0, 5000},
	}
	for _, c := range cases {
		s, err := SplitPayment(c.amount, c.rate)
		if err != nil {
			t.Fatal(err)
		}
		if s.PlatformFee != c.fee || s.WorkerAmount != c.worker || s.PlatformFee+s.WorkerAmount != c.amount {
			t.Errorf("SplitPayment(%d, %v) = %+v, want fee %d worker %d", c.amount, c.rate, s, c.fee, c.worker)
		}
	}
	if _, err := SplitPayment(100, 1.5); err == nil {
		t.Error("expected error for rate > 1")
	}
}

func TestPartialRefund(t *testing.T) {
	s, err := PartialRefund(10000, 4000, 0.10)
	if err != nil {
		t.Fatal(err)
	}
	if s.Amount != 6000 || s.PlatformFee != 600 || s.WorkerAmount != 5400 {
		t.Errorf("got %+v", s)
	}
	if _, err := PartialRefund(10000, 12000, 0.1); err == nil {
		t.Error("expected error for refund > amount")
	}
}

func TestTransitions(t *testing.T) {
	ok := []struct{ from, to JobStatus }{
		{StatusAssigned, StatusWorkerOnTheWay},
		{StatusWorkerOnTheWay, StatusArrived},
		{StatusArrived, StatusStarted},
		{StatusAssigned, StatusStarted}, // remote/digital jobs skip travel
		{StatusStarted, StatusAwaitingConfirmation},
	}
	for _, c := range ok {
		if !CanAdvance(c.from, c.to, ActorWorker) {
			t.Errorf("worker should advance %s → %s", c.from, c.to)
		}
		if CanAdvance(c.from, c.to, ActorClient) {
			t.Errorf("client must not advance %s → %s", c.from, c.to)
		}
	}
	bad := []struct{ from, to JobStatus }{
		{StatusMatching, StatusStarted},
		{StatusAssigned, StatusAwaitingConfirmation},
		{StatusStarted, StatusCompleted}, // only the client confirms
		{StatusCompleted, StatusStarted},
	}
	for _, c := range bad {
		if CanAdvance(c.from, c.to, ActorWorker) {
			t.Errorf("worker must not advance %s → %s", c.from, c.to)
		}
	}
}

func TestCancelRules(t *testing.T) {
	if !CanCancel(StatusMatching, ActorClient) || CanCancel(StatusStarted, ActorClient) {
		t.Error("client cancel rules wrong")
	}
	if CanCancel(StatusMatching, ActorWorker) || !CanCancel(StatusAssigned, ActorWorker) {
		t.Error("worker cancel rules wrong")
	}
	if CancellationHurtsReputation(StatusMatching, ActorClient) {
		t.Error("cancelling an open job should be free")
	}
	if !CancellationHurtsReputation(StatusAssigned, ActorWorker) {
		t.Error("worker dropping a job should count")
	}
}

func TestDistance(t *testing.T) {
	// Cocody Riviera → Plateau (Abidjan) is roughly 7 km.
	d := DistanceKm(5.3600, -3.9600, 5.3200, -4.0200)
	if d < 6 || d > 9 {
		t.Errorf("unexpected distance %.2f", d)
	}
	if DistanceKm(5.36, -3.96, 5.36, -3.96) != 0 {
		t.Error("zero distance expected")
	}
	if RoundDistance(1.84) != 1.8 {
		t.Error("rounding")
	}
}

func TestMatchScore(t *testing.T) {
	near := MatchScore(MatchInput{DistanceKm: 1, RadiusKm: 10})
	far := MatchScore(MatchInput{DistanceKm: 9, RadiusKm: 10})
	if near <= far {
		t.Error("closer worker should score higher")
	}
	oneFiveStar := MatchScore(MatchInput{DistanceKm: 2, RadiusKm: 10, Rating: 5, RatingCount: 1})
	established := MatchScore(MatchInput{DistanceKm: 2, RadiusKm: 10, Rating: 4.8, RatingCount: 120, SimilarJobs: 30})
	if oneFiveStar >= established {
		t.Errorf("single 5★ (%.3f) should not beat established 4.8★ (%.3f)", oneFiveStar, established)
	}
	if s := MatchScore(MatchInput{DistanceKm: 50, RadiusKm: 10}); s < 0 || math.IsNaN(s) {
		t.Error("score must stay non-negative")
	}
}
