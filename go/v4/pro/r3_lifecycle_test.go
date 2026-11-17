package ccxtpro

// Regression tests (R3 review): WS lifecycle leaks around Exchange.Close / Shutdown.

import (
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/gorilla/websocket"
)

func r3Stacks(needle string) int {
	buf := make([]byte, 1<<24)
	buf = buf[:runtime.Stack(buf, true)]
	n := 0
	for _, g := range strings.Split(string(buf), "\n\n") {
		if strings.Contains(g, needle) {
			n++
		}
	}
	return n
}

// holeURL accepts TCP and never answers the WS handshake; it closes each socket after hold.
func holeURL(t *testing.T, hold time.Duration) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() { time.Sleep(hold); c.Close() }()
		}
	}()
	return "ws://" + l.Addr().String()
}

func newProbeExchange(timeoutMs int) *ccxt.Exchange {
	e := NewDeribit(nil)
	ex := &e.Core.Exchange
	ex.HttpProxy = "unused" // lets Exchange.Client build a ws:// client; the agent is not used by the dial
	if timeoutMs > 0 {
		ex.Options.Store("ws", map[string]any{"ConnectionTimeout": timeoutMs})
	}
	return ex
}

func watchN(ex *ccxt.Exchange, url string, n int) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			h := "h" + string(rune('a'+i))
			msg := map[string]any{"sub": h}
			select {
			case <-ex.Watch(url, h, msg, h, msg):
			case <-time.After(5 * time.Second):
			}
		}(i)
	}
	wg.Wait()
}

// L1: the connection timer fires before the dial fails, evicting the client; a second watcher then parks
// on the never-settled Connected future, and Exchange.Close (empty cache) never reaches it.
func TestR3TimedOutDialParksWatcherForever(t *testing.T) {
	url := holeURL(t, 1*time.Second)
	ex := newProbeExchange(300)
	watchN(ex, url, 2)
	ex.Shutdown()
	time.Sleep(1500 * time.Millisecond)
	if n := r3Stacks("(*Exchange).Watch.func"); n > 0 {
		t.Errorf("L1: %d Watch goroutines still parked after Exchange.Close (each pins the client and the exchange)", n)
	}
}

// F-A makes the parked goroutines' error branch live: Close rejects Connected and every parked watcher
// deletes from client.Subscriptions without SubscriptionsMu.
func TestR3CloseWakesParkedWatchersConcurrently(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	url := "ws://" + l.Addr().String()
	l.Close() // refused at once: the first dial fails, the rest park on Connected
	for round := 0; round < 20; round++ {
		ex := newProbeExchange(0)
		watchN(ex, url, 12)
		ex.Close()
	}
	time.Sleep(200 * time.Millisecond)
}

// L2: Exchange has no closed state; a Watch that reaches Client(url) after Close dials a live socket that
// nothing will ever close.
func TestR3WatchAfterShutdownDialsNothing(t *testing.T) {
	up := websocket.Upgrader{}
	open := make(chan struct{}, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		open <- struct{}{}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	ex := newProbeExchange(0)
	ex.Shutdown()                                                                                   // the generation retired
	go func() { <-ex.Watch(url, "late", map[string]any{"x": 1}, "late", map[string]any{"x": 1}) }() // an abandoned in-flight watch
	select {
	case <-open:
		t.Errorf("L2: a watch after Exchange.Shutdown opened a live socket on the closed exchange")
	case <-time.After(2 * time.Second):
	}
}

// Close keeps its re-dial semantics for owners that reuse the exchange (the crypto poller).
func TestCloseThenWatchStillRedials(t *testing.T) {
	up := websocket.Upgrader{}
	open := make(chan struct{}, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		open <- struct{}{}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	ex := newProbeExchange(0)
	ex.Close()
	go func() { <-ex.Watch(url, "again", map[string]any{"x": 1}, "again", map[string]any{"x": 1}) }()
	select {
	case <-open:
	case <-time.After(3 * time.Second):
		t.Errorf("a watch after Close must still dial (Close is not permanent)")
	}
	ex.Shutdown()
}
