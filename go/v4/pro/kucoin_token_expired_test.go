package ccxtpro

// AGT-861: an error frame that a handler raises as an exception must reject the client's watchers,
// not panic the read-loop goroutine (which nothing can recover, so it killed the process).

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/gorilla/websocket"
)

// errorFrameURL serves a WS endpoint that answers every subscribe with frame.
func errorFrameURL(t *testing.T, frame string) string {
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
			if err := c.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func TestKucoinTokenExpiredRejectsWatcher(t *testing.T) {
	for _, connectID := range []string{"public", "private", "publicFutures", "privateFutures"} {
		t.Run(connectID, func(t *testing.T) {
			url := errorFrameURL(t, `{"id":"1","type":"error","code":401,"data":"token is expired"}`) + "?connectId=" + connectID
			k := NewKucoin(nil)
			ex := &k.Core.Exchange
			ex.HttpProxy = "unused" // lets Exchange.Client build a ws:// client; the agent is not used by the dial
			defer ex.Shutdown()
			// Stands in for the urls Negotiate caches per connection; options.urls is unset until then.
			urls := map[string]any{}
			for _, id := range []string{"public", "private", "publicFutures", "privateFutures"} {
				urls[id] = "cached-" + id
			}
			ccxt.AddElementToObject(k.Core.Options, "urls", urls)

			msg := map[string]any{"id": "1", "type": "subscribe", "topic": "/market/level2:BTC-USDT"}
			select {
			case res := <-ex.Watch(url, "orderbook:BTC/USDT", msg, "orderbook:BTC/USDT", msg):
				err, ok := res.(error)
				if !ok {
					t.Fatalf("watch resolved with %T %v, want an error", res, res)
				}
				if !strings.Contains(err.Error(), "token is expired") {
					t.Errorf("watch error = %v, want the venue's token-expired message", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("watch never settled after the venue's error frame")
			}
			// Only the expired connection's cached url is cleared, so its next watch negotiates a fresh token.
			for _, id := range []string{"public", "private", "publicFutures", "privateFutures"} {
				got := urls[id]
				if id == connectID && got != nil {
					t.Errorf("%s url = %v, want it cleared", id, got)
				}
				if id != connectID && got == nil {
					t.Errorf("%s url cleared, want only %s cleared", id, connectID)
				}
			}
		})
	}
}
