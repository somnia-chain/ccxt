package ccxt

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// A watch call can look up a client just before its connection drops and
// create its future just after Reset has rejected the pending ones. The dead
// client is never re-dialled and the symbols already count as subscribed, so
// nothing resolves or rejects that future: the watch hangs (AGT-781).

// newDroppingWSServer accepts WebSocket upgrades and severs each connection's
// TCP socket, without a close frame, when drop is closed — how binance's
// abnormal 1006 closures reach the client.
func newDroppingWSServer(t *testing.T) (url string, drop func()) {
	t.Helper()
	dropCh := make(chan struct{})
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		<-dropCh
		conn.UnderlyingConn().Close()
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http"), func() { close(dropCh) }
}

func TestFutureOnDroppedClientIsRejected(t *testing.T) {
	url, drop := newDroppingWSServer(t)
	ex := &Exchange{Clients: map[string]any{}}
	client, done := connectTracked(t, ex, url)

	drop()
	awaitReadLoop(t, done)

	fut := client.NewFuture("bidask:BTC/USDT:USDT")
	select {
	case res := <-fut.Await():
		if _, ok := res.(error); !ok {
			t.Fatalf("future on a dead client resolved with %v, want an error", res)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("future on a dead client never settled")
	}
}

func TestFutureRaceOnDroppedClientIsRejected(t *testing.T) {
	url, drop := newDroppingWSServer(t)
	ex := &Exchange{Clients: map[string]any{}}
	client, done := connectTracked(t, ex, url)

	drop()
	awaitReadLoop(t, done)

	race := FutureRace([]*Future{client.NewFuture("bidask:BTC/USDT:USDT"), client.NewFuture("bidask:ETH/USDT:USDT")})
	select {
	case res := <-race.Await():
		if _, ok := res.(error); !ok {
			t.Fatalf("race on a dead client resolved with %v, want an error", res)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("race on a dead client never settled")
	}
}

// A future created on a healthy client must stay pending until data arrives.
func TestFutureOnLiveClientStaysPending(t *testing.T) {
	url, _ := newDroppingWSServer(t)
	ex := &Exchange{Clients: map[string]any{}}
	client, _ := connectTracked(t, ex, url)

	fut := client.NewFuture("bidask:BTC/USDT:USDT")
	select {
	case res := <-fut.Await():
		t.Fatalf("future on a live client settled early with %v", res)
	case <-time.After(200 * time.Millisecond):
	}
}
