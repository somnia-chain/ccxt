package ccxt

import (
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func closedPortURL(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	return "ws://" + addr
}

func awaitWithin(t *testing.T, f *Future, d time.Duration, what string) {
	t.Helper()
	select {
	case <-f.Await():
	case <-time.After(d):
		t.Fatalf("%s did not return within %v", what, d)
	}
}

func newTestWS(url string) *WSClient {
	nop := func(client any, err any) {}
	return NewWSClient(url, nop, nop, nop, nop, "")
}

func TestWSClientCloseNeverConnectedReturns(t *testing.T) {
	c := newTestWS(closedPortURL(t))
	if _, err := c.Connect(); err == nil {
		t.Fatal("expected dial error")
	}
	awaitWithin(t, c.Close(), time.Second, "Close().Await()")
}

func TestExchangeCloseNeverConnectedReturns(t *testing.T) {
	url := closedPortURL(t)
	c := newTestWS(url)
	if _, err := c.Connect(); err == nil {
		t.Fatal("expected dial error")
	}
	ex := &Exchange{Clients: map[string]any{url: c}}
	done := make(chan struct{})
	go func() { ex.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Exchange.Close did not return within 1s")
	}
}

func TestCloseDuringDialDoesNotInstallConnection(t *testing.T) {
	base := runtime.NumGoroutine()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	serverEOF := make(chan struct{})
	up := websocket.Upgrader{}
	go func() {
		srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond)
			conn, err := up.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					close(serverEOF)
					return
				}
			}
		})}
		srv.Serve(l)
	}()
	c := newTestWS("ws://" + l.Addr().String())
	dialDone := make(chan error, 1)
	go func() { _, err := c.Connect(); dialDone <- err }()
	time.Sleep(100 * time.Millisecond)
	awaitWithin(t, c.Close(), time.Second, "Close().Await()")
	select {
	case <-dialDone:
	case <-time.After(3 * time.Second):
		t.Fatal("dial did not finish")
	}
	if c.IsOpen() {
		t.Fatal("connection installed after Close")
	}
	select {
	case <-serverEOF:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not see EOF")
	}
	l.Close()
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > base+2 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > base+2 {
		t.Fatalf("goroutines %d, baseline %d", n, base)
	}
}

func TestCloseConnectedResolvesViaReadLoop(t *testing.T) {
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	c := newTestWS("ws" + strings.TrimPrefix(srv.URL, "http"))
	f, err := c.Connect()
	if err != nil {
		t.Fatal(err)
	}
	awaitWithin(t, f, 2*time.Second, "Connected")
	awaitWithin(t, c.Close(), 2*time.Second, "Close().Await()")
	awaitWithin(t, c.Close(), time.Second, "second Close().Await()")
}
