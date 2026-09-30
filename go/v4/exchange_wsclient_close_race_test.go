package ccxt

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// These tests cover Exchange.Close racing the WSClient read-loop goroutine
// started by CreateConnection. Closing the socket makes ReadMessage fail, so
// the read loop runs OnError -> Reset -> Client.Close and then its deferred
// OnClose, concurrently with Exchange.Close still tearing the client down.
// Run with -race: the failures are data races on Client.Connection and
// Client.Error, not assertion failures.

// newIdleWSServer accepts WebSocket upgrades and holds each connection open
// without writing, so the read loop blocks in ReadMessage until closed.
func newIdleWSServer(t *testing.T) string {
	t.Helper()
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// connectTracked wires a WSClient to ex's callbacks the way Exchange.Client
// does, registers it, connects it, and returns a channel closed once the read
// loop's OnClose has run.
func connectTracked(t *testing.T, ex *Exchange, url string) (*WSClient, <-chan struct{}) {
	t.Helper()
	readLoopDone := make(chan struct{})
	onClose := func(client any, err any) {
		ex.OnClose(client, err)
		close(readLoopDone)
	}
	client := NewWSClient(url, func(any, any) {}, ex.OnError, onClose, ex.OnConnected, "")
	ex.WsClientsMu.Lock()
	ex.Clients[url] = client
	ex.WsClientsMu.Unlock()
	if err := client.CreateConnection(); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	select {
	case <-client.Connected.(*Future).Await():
	case <-time.After(5 * time.Second):
		t.Fatal("client never reported connected")
	}
	return client, readLoopDone
}

func awaitReadLoop(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("read loop did not exit after Close")
	}
}

func TestExchangeCloseDoesNotRaceReadLoop(t *testing.T) {
	url := newIdleWSServer(t)
	for i := 0; i < 20; i++ {
		ex := &Exchange{Clients: map[string]any{}}
		_, done := connectTracked(t, ex, url)

		if errs := ex.Close(); len(errs) != 0 {
			t.Fatalf("Close returned errors: %v", errs)
		}
		awaitReadLoop(t, done)
	}
}

// After Close the client must read as closed and carry the error that caused
// it, whichever goroutine won the teardown.
func TestExchangeCloseLeavesClientClosedWithError(t *testing.T) {
	url := newIdleWSServer(t)
	ex := &Exchange{Clients: map[string]any{}}
	client, done := connectTracked(t, ex, url)

	ex.Close()
	awaitReadLoop(t, done)

	if client.IsOpen() {
		t.Error("client still open after Exchange.Close")
	}
	if client.GetError() == nil {
		t.Error("client has no error after Exchange.Close")
	}
}
