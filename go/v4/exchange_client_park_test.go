package ccxt

import (
	"fmt"
	"testing"
	"time"
)

func newParkTestClient(subscribed ...string) *Client {
	c := &Client{Futures: map[string]any{}, Subscriptions: map[string]any{}, Rejections: map[string]any{}}
	for _, h := range subscribed {
		c.Subscriptions[h] = make(chan any)
	}
	return c
}

// watchOnce is one WatchMultiple call as the generated code makes it: a future
// per hash, raced, awaited.
func watchOnce(t *testing.T, c *Client, hashes []string) any {
	t.Helper()
	futures := make([]*Future, len(hashes))
	for i, h := range hashes {
		futures[i] = c.NewFuture(h)
	}
	select {
	case v := <-FutureRace(futures).Await():
		return v
	case <-time.After(time.Second):
		return nil
	}
}

// An update for a subscribed hash that lands while no call is waiting is kept
// for the next call rather than dropped.
func TestResolveWithNoWaiterIsDeliveredToNextCall(t *testing.T) {
	c := newParkTestClient("ticker::A")
	c.Resolve("a1", "ticker::A")

	if got := watchOnce(t, c, []string{"ticker::A"}); got != "a1" {
		t.Fatalf("next call got %v, want a1", got)
	}
	if _, parked := c.Futures["ticker::A"]; parked {
		t.Fatal("delivered value still parked")
	}
}

// A hash nobody subscribed to (a one-shot request id) is not retained.
func TestResolveForUnsubscribedHashIsNotRetained(t *testing.T) {
	c := newParkTestClient()
	c.Resolve("ack", "req::1")

	if len(c.Futures) != 0 {
		t.Fatalf("Futures = %v, want empty", c.Futures)
	}
}

// Only the newest undelivered value per hash is kept.
func TestParkedValueIsReplacedByNewer(t *testing.T) {
	c := newParkTestClient("ticker::A")
	c.Resolve("a1", "ticker::A")
	c.Resolve("a2", "ticker::A")

	if got := watchOnce(t, c, []string{"ticker::A"}); got != "a2" {
		t.Fatalf("next call got %v, want a2", got)
	}
}

// The losing futures of a race stay in the map with no one listening; an
// update resolving one of them must still reach a later call.
func TestUpdateForRaceLoserReachesNextCall(t *testing.T) {
	c := newParkTestClient("ticker::A", "ticker::B")
	hashes := []string{"ticker::A", "ticker::B"}

	done := make(chan any, 1)
	go func() { done <- watchOnce(t, c, hashes) }()
	waitForSubscribers(t, c, "ticker::A")
	c.Resolve("a1", "ticker::A")
	if got := <-done; got != "a1" {
		t.Fatalf("first call got %v, want a1", got)
	}

	c.Resolve("b1", "ticker::B")
	if got := watchOnce(t, c, hashes); got != "b1" {
		t.Fatalf("second call got %v, want b1", got)
	}
}

// A burst across many hashes between calls is drained one per call, none lost.
func TestBurstBetweenCallsIsFullyDelivered(t *testing.T) {
	hashes := make([]string, 20)
	for i := range hashes {
		hashes[i] = fmt.Sprintf("ticker::S%02d", i)
	}
	c := newParkTestClient(hashes...)
	for i, h := range hashes {
		c.Resolve(i, h)
	}

	seen := map[any]bool{}
	for range hashes {
		seen[watchOnce(t, c, hashes)] = true
	}
	for i := range hashes {
		if !seen[i] {
			t.Errorf("update %d never delivered", i)
		}
	}
}

// Updates racing a live call are each delivered exactly once across calls.
func TestConcurrentUpdatesAreEachDelivered(t *testing.T) {
	hashes := make([]string, 20)
	for i := range hashes {
		hashes[i] = fmt.Sprintf("ticker::S%02d", i)
	}
	c := newParkTestClient(hashes...)

	go func() {
		for i, h := range hashes {
			c.Resolve(i, h)
		}
	}()
	seen := map[any]int{}
	for range hashes {
		seen[watchOnce(t, c, hashes)]++
	}
	for i := range hashes {
		if seen[i] != 1 {
			t.Errorf("update %d delivered %d times, want 1", i, seen[i])
		}
	}
}

func waitForSubscribers(t *testing.T, c *Client, hash string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		c.FuturesMu.RLock()
		f, ok := c.Futures[hash].(*Future)
		c.FuturesMu.RUnlock()
		if ok {
			f.subscribersMu.Lock()
			n := len(f.subscribers)
			f.subscribersMu.Unlock()
			if n > 0 {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("no subscriber on %s", hash)
}
