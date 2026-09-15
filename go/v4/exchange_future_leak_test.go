package ccxt

import (
	"errors"
	"testing"
)

// These tests cover subscriber-slice retention in FutureRace, which is distinct
// from the goroutine growth covered by exchange_future_race_test.go (#28182).
//
// Client.NewFuture reuses the *Future stored under a messageHash, and
// Client.Resolve deletes only the hash that actually fired. So across repeated
// WatchMultiple calls the futures for quiet symbols are the same objects every
// time, while the winner is replaced. Any per-call state FutureRace leaves on a
// losing future therefore accumulates for the lifetime of the client.

// raceOver races a fresh winner against the supplied long-lived futures and
// waits for the race to settle, mimicking one WatchMultiple call.
func raceOver(longLived []*Future, resolve func(*Future)) {
	winner := NewFuture()
	all := make([]*Future, 0, len(longLived)+1)
	all = append(all, longLived...)
	all = append(all, winner)

	race := FutureRace(all)
	resolve(winner)
	<-race.Await()
}

func subscriberCount(f *Future) int {
	f.subscribersMu.Lock()
	defer f.subscribersMu.Unlock()
	return len(f.subscribers)
}

// A future whose messageHash never fires must not accumulate one shared
// channel per race. Each retained channel is buffered, so a later resolution
// deposits a full parsed value into every one of them — that is the leak.
func TestFutureRaceUnsubscribesLosersAfterSettle(t *testing.T) {
	const iterations = 100

	quiet := []*Future{NewFuture(), NewFuture()}

	for i := 0; i < iterations; i++ {
		raceOver(quiet, func(w *Future) { w.Resolve("tick") })
	}

	for i, f := range quiet {
		if n := subscriberCount(f); n != 0 {
			t.Errorf("quiet future %d retained %d subscriber channels after %d races; want 0",
				i, n, iterations)
		}
	}
}

// Same requirement when the race settles by rejection rather than resolution.
func TestFutureRaceUnsubscribesLosersAfterRejection(t *testing.T) {
	const iterations = 50

	quiet := []*Future{NewFuture()}

	for i := 0; i < iterations; i++ {
		raceOver(quiet, func(w *Future) { w.Reject(errors.New("boom")) })
	}

	if n := subscriberCount(quiet[0]); n != 0 {
		t.Errorf("quiet future retained %d subscriber channels after %d rejected races; want 0",
			n, iterations)
	}
}

// When FutureRace encounters an already-resolved future partway through the
// slice it returns immediately, without ever starting the goroutine that reads
// the shared channel. Futures subscribed before that point are left holding a
// channel nothing will ever read.
func TestFutureRaceUnsubscribesWhenPrecompletedFutureShortCircuits(t *testing.T) {
	const iterations = 50

	quiet := NewFuture()

	for i := 0; i < iterations; i++ {
		ready := NewFuture()
		ready.Resolve("already done")
		<-FutureRace([]*Future{quiet, ready}).Await()
	}

	if n := subscriberCount(quiet); n != 0 {
		t.Errorf("future subscribed before the short-circuit retained %d subscriber channels "+
			"after %d races; want 0", n, iterations)
	}
}

// Same short-circuit path, reached via a pre-rejected future.
func TestFutureRaceUnsubscribesWhenPrerejectedFutureShortCircuits(t *testing.T) {
	quiet := NewFuture()

	ready := NewFuture()
	ready.Reject(errors.New("already failed"))

	<-FutureRace([]*Future{quiet, ready}).Await()

	if n := subscriberCount(quiet); n != 0 {
		t.Errorf("future subscribed before the short-circuit retained %d subscriber channels; want 0", n)
	}
}

// The winner clears its own subscribers in Resolve; deregistration must not
// disturb that, nor panic on a future that has already emptied its slice.
func TestFutureRaceWinnerSubscribersStillCleared(t *testing.T) {
	quiet := NewFuture()
	winner := NewFuture()

	race := FutureRace([]*Future{quiet, winner})
	winner.Resolve("value")
	<-race.Await()

	if n := subscriberCount(winner); n != 0 {
		t.Errorf("winner retained %d subscriber channels; want 0", n)
	}
	if n := subscriberCount(quiet); n != 0 {
		t.Errorf("loser retained %d subscriber channels; want 0", n)
	}
}

// Deregistering the settled race's channel must not disturb an unrelated
// concurrent race still waiting on the same futures.
func TestFutureRaceDeregistrationLeavesConcurrentRaceIntact(t *testing.T) {
	shared := NewFuture()

	firstWinner := NewFuture()
	first := FutureRace([]*Future{shared, firstWinner})

	second := FutureRace([]*Future{shared})

	firstWinner.Resolve("first")
	<-first.Await()

	// The second race is still pending and must still be subscribed to shared.
	if n := subscriberCount(shared); n != 1 {
		t.Fatalf("shared future has %d subscribers after the first race settled; want 1 "+
			"(the still-pending second race)", n)
	}

	shared.Resolve("second")
	if got := <-second.Await(); got != "second" {
		t.Errorf("second race resolved to %v; want \"second\"", got)
	}
}
