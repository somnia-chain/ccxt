package ccxt

import (
	"sync"
	"sync/atomic"
)

// Future is a one-shot promise
//	- once resolved or rejected its channel is closed and subsequent Resolve/Reject calls are ignored.
//	- matches the minimal API the generated WS code expects: Resolve(...).
//	- use the channel returned by Await() (or the struct itself) to receive the value

type GetsLimit interface {
	GetLimit(symbol any, limit any) any
}

// used when a value does not implement GetsLimit
// returns the caller-supplied limit unchanged
type NoopLimit struct{ Val any }

func (n NoopLimit) GetLimit(symbol any, limit any) any { return limit }

// converts arbitrary values to the GetsLimit interface expected by Future.Resolve
func ToGetsLimit(v any) GetsLimit {
	if gl, ok := v.(GetsLimit); ok {
		//If the value already implements GetsLimit it is returned verbatim
		return gl
	}
	return NoopLimit{Val: v} // otherwise it is wrapped in NoopLimit
}

type Future struct {
	result        chan any
	err           chan any
	subscribers   []subscriber
	resolved      bool
	resolvedValue any
	resolvedError any
	onConsume     func() // set while the future is parked on a Client; see Client.Resolve
	mu            sync.Mutex
	once          sync.Once
	subscribersMu sync.Mutex
}

// subscriber is one waiter on a Future. FutureRace shares one taken flag
// across every future it subscribes to, so exactly one of them hands the race
// its value; the rest see the flag set and know nobody is waiting on them.
type subscriber struct {
	ch    chan any
	taken *atomic.Bool
}

func newSubscriber(ch chan any) subscriber { return subscriber{ch: ch, taken: new(atomic.Bool)} }

// waited reports whether a subscriber is still waiting for this future.
func (f *Future) waited() bool {
	f.subscribersMu.Lock()
	defer f.subscribersMu.Unlock()
	for _, sub := range f.subscribers {
		if !sub.taken.Load() {
			return true
		}
	}
	return false
}

func (f *Future) isResolved() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.resolved
}

// catchUp delivers to sub if f was resolved between the caller checking and
// sub being added, a window in which Resolve would have missed it.
func (f *Future) catchUp(sub subscriber) bool {
	f.mu.Lock()
	resolved, value, reason := f.resolved, f.resolvedValue, f.resolvedError
	f.mu.Unlock()
	if !resolved || !sub.taken.CompareAndSwap(false, true) {
		return false
	}
	if reason != nil {
		value = reason
	}
	select {
	case sub.ch <- value:
	default:
	}
	return true
}

// consume detaches a parked future from its client once its value has been
// handed to a caller, so the next call waits for a fresh update.
func (f *Future) consume() {
	f.mu.Lock()
	detach := f.onConsume
	f.onConsume = nil
	f.mu.Unlock()
	if detach != nil {
		detach()
	}
}

// Create new Future
func NewFuture() *Future {
	return &Future{
		result: make(chan any, 1),
		err:    make(chan any, 1),
	}
}

// Resolve asynchronously with a value
func (f *Future) Resolve(args ...any) {
	var value any
	if len(args) == 0 {
		value = nil
	} else {
		value = args[0]
	}
	f.once.Do(func() {
		f.mu.Lock()
		f.resolved = true
		f.resolvedValue = value
		f.resolvedError = nil
		f.mu.Unlock()

		func() {
			defer func() {
				if r := recover(); r != nil {
					// Channel is closed, but that's okay since we're using sync.Once
					// and the future is already marked as resolved
				}
			}()
			select {
			case f.result <- value:
			default:
			}
		}()

		f.subscribersMu.Lock()
		// Notify all subscribers
		for _, sub := range f.subscribers {
			if !sub.taken.CompareAndSwap(false, true) {
				continue
			}
			func(sub chan any) {
				defer func() {
					if r := recover(); r != nil {
						// Channel is closed, but that's okay since we're using sync.Once
						// and the future is already marked as resolved
					}
				}()
				select {
				case sub <- value:
				default:
				}
			}(sub.ch)
		}
		f.subscribers = nil // Clear subscribers after notifying them
		f.subscribersMu.Unlock()
	})
}

// Reject asynchronously with an error
func (f *Future) Reject(reason any) {
	f.once.Do(func() {
		f.mu.Lock()
		f.resolved = true
		f.resolvedValue = nil
		f.resolvedError = reason
		f.mu.Unlock()

		func() {
			defer func() {
				if r := recover(); r != nil {
					// Channel is closed, but that's okay since we're using sync.Once
					// and the future is already marked as resolved
				}
			}()
			select {
			case f.err <- reason:
			default:
			}
		}()

		// Notify all subscribers
		f.subscribersMu.Lock()
		for _, sub := range f.subscribers {
			if !sub.taken.CompareAndSwap(false, true) {
				continue
			}
			func(sub chan any) {
				defer func() {
					if r := recover(); r != nil {
						// Channel is closed, but that's okay since we're using sync.Once
						// and the future is already marked as resolved
					}
				}()
				select {
				case sub <- reason:
				default:
				}
			}(sub.ch)
		}
		f.subscribers = nil // Clear subscribers after notifying them
		f.subscribersMu.Unlock()
	})
}

// // Await blocks until either result or error is received
// // Returns the resolved value (which could be an error)
// func (f *Future) Await() <-chan any {
// 	ch := make(chan any)

// 	go func() {
// 		defer close(ch)

// 		// f.mu.Lock()
// 		if f.resolved {
// 			// f.mu.Unlock()
// 			// for {
// 			// Already resolved, return cached value immediately
// 			// f.mu.Lock()
// 			if f.resolvedError != nil {
// 				ch <- f.resolvedError
// 			} else {
// 				ch <- f.resolvedValue
// 			}
// 			// f.mu.Unlock()
// 			// f.mu.Unlock()
// 			// return
// 		}
// 		// }
// 		// f.mu.Unlock()

// 		// // Not resolved yet, wait for it
// 		// select {
// 		// case res := <-f.result:
// 		// 	for {
// 		// 		ch <- res
// 		// 	}
// 		// case err := <-f.err:
// 		// 	for {
// 		// 		ch <- err
// 		// 	}
// 		// }

// 		resCh, errCh := f.result, f.err
// 		// f.mu.Unlock()

// 		var out any
// 		select {
// 		case out = <-resCh:
// 		case out = <-errCh:
// 		}

// 		// Cache the resolution
// 		f.mu.Lock()
// 		f.resolved = true
// 		if e, ok := out.(error); ok {
// 			f.resolvedError = e
// 		} else {
// 			f.resolvedValue = out
// 		}
// 		val, err := f.resolvedValue, f.resolvedError
// 		f.mu.Unlock()

// 		// for {
// 		if err != nil {
// 			ch <- err
// 		} else {
// 			ch <- val
// 		}
// 		// }
// 	}()

// 	return ch
// }

func (f *Future) Await() <-chan any {
	ch := make(chan any, 1)
	f.mu.Lock()
	if f.resolved {
		// Already resolved, return cached value immediately
		if f.resolvedError != nil {
			ch <- f.resolvedError
		} else {
			ch <- f.resolvedValue
		}
		f.mu.Unlock()
		f.consume()
		return ch
	}
	f.mu.Unlock()
	sub := newSubscriber(ch)
	f.subscribersMu.Lock()
	if f.subscribers == nil {
		f.subscribers = make([]subscriber, 0)
	}
	f.subscribers = append(f.subscribers, sub)
	f.subscribersMu.Unlock()
	if f.catchUp(sub) {
		f.consume()
	}
	// go func() {
	// 	defer close(ch)
	// 	// f.mu.Lock()
	// 	if f.resolved {
	// 		// Already resolved, return cached value immediately
	// 		if f.resolvedError != nil {
	// 			ch <- f.resolvedError
	// 		} else {
	// 			ch <- f.resolvedValue
	// 		}
	// 		// f.mu.Unlock()
	// 		return
	// 	}

	// 	// f.mu.Unlock()

	// 	// Not resolved yet, wait for it
	// 	select {
	// 	case res := <-f.result:
	// 		ch <- res
	// 	case err := <-f.err:
	// 		ch <- err
	// 	}
	// }()

	return ch
}

// Wrap an existing channel that returns (any, error) into Future
func WrapFuture(ch <-chan struct {
	val any
	err error
}) *Future {
	f := NewFuture()
	go func() {
		v := <-ch
		if v.err != nil {
			f.Reject(v.err)
		} else {
			f.Resolve(v.val)
		}
	}()
	return f
}

// unsubscribe removes ch from f's subscriber list. It is a no-op when ch is
// absent, which is the normal case for the future that won a race: Resolve and
// Reject already cleared the whole slice.
func (f *Future) unsubscribe(ch chan interface{}) {
	f.subscribersMu.Lock()
	defer f.subscribersMu.Unlock()
	for i, sub := range f.subscribers {
		if sub.ch == ch {
			f.subscribers = append(f.subscribers[:i], f.subscribers[i+1:]...)
			return
		}
	}
}

// Race multiple Futures: returns the first resolved or rejected value/error.
// Uses a shared subscriber channel instead of one goroutine per future to
// avoid O(N) goroutine creation on every call (fixes #28182).
//
// Once the race settles, the shared channel is removed from every future that
// did not win. Futures are long-lived and reused per messageHash (see
// Client.NewFuture) while Client.Resolve deletes only the hash that actually
// fired, so a symbol whose hash rarely fires stays in the map across calls.
// Without deregistration it would retain one channel per race for the lifetime
// of the client, and Resolve's non-blocking send would later deposit a full
// resolved value into each of those buffers.
func FutureRace(futures []*Future) *Future {
	result := NewFuture()
	// Buffered so that a non-blocking send from Future.Resolve succeeds
	// even before the reader goroutine is scheduled.
	sharedCh := make(chan interface{}, 1)
	taken := new(atomic.Bool)

	// Tracks the futures that actually got sharedCh appended, so the
	// short-circuit below can undo the subscriptions made before it.
	subscribed := make([]*Future, 0, len(futures))
	unsubscribeAll := func() {
		for _, f := range subscribed {
			f.unsubscribe(sharedCh)
		}
	}

	for _, f := range futures {
		f.mu.Lock()
		if f.resolved {
			val, err := f.resolvedValue, f.resolvedError
			f.mu.Unlock()
			f.consume()
			// Returning here skips the forwarding goroutine, so nothing would
			// ever drain sharedCh from the futures already subscribed above.
			unsubscribeAll()
			if err != nil {
				result.Reject(err.(error))
			} else {
				result.Resolve(val)
			}
			return result
		}
		f.mu.Unlock()

		f.subscribersMu.Lock()
		if f.subscribers == nil {
			f.subscribers = make([]subscriber, 0)
		}
		sub := subscriber{ch: sharedCh, taken: taken}
		f.subscribers = append(f.subscribers, sub)
		f.subscribersMu.Unlock()
		if f.catchUp(sub) {
			f.consume()
		}
		subscribed = append(subscribed, f)
	}

	// Single goroutine forwards the first resolved/rejected value. Deregister
	// before settling `result` so that a caller awaiting it observes the
	// cleanup as already done.
	go func() {
		val := <-sharedCh
		unsubscribeAll()
		if err, isError := val.(error); isError {
			result.Reject(err)
		} else {
			result.Resolve(val)
		}
	}()

	return result
}
