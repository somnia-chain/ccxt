package ccxt

import (
	"net/http"
	"sync"
	"testing"
)

// UpdateProxySettings is called from Fetch on every single request. Rebuilding
// the http.Transport each time orphans the previous one together with its
// parked idle connections, and defeats connection pooling entirely.

func newProxyTestExchange(proxy string) *Exchange {
	ex := &Exchange{}
	ex.Init(map[string]any{})
	ex.httpClient = &http.Client{Transport: &http.Transport{}}
	ex.HttpProxy = proxy
	return ex
}

func TestUpdateProxySettingsReusesTransportWhenProxyUnchanged(t *testing.T) {
	ex := newProxyTestExchange("http://127.0.0.1:3128")

	ex.UpdateProxySettings()
	first := ex.httpClient.Transport

	for i := 0; i < 10; i++ {
		ex.UpdateProxySettings()
	}

	if got := ex.httpClient.Transport; got != first {
		t.Errorf("Transport was replaced across repeated calls with an unchanged proxy; "+
			"want the same instance reused (first=%p last=%p)", first, got)
	}
}

func TestUpdateProxySettingsRebuildsTransportWhenProxyChanges(t *testing.T) {
	ex := newProxyTestExchange("http://127.0.0.1:3128")

	ex.UpdateProxySettings()
	first := ex.httpClient.Transport

	ex.HttpProxy = "http://127.0.0.1:9999"
	ex.UpdateProxySettings()

	if got := ex.httpClient.Transport; got == first {
		t.Error("Transport was reused after the proxy URL changed; want a rebuilt transport")
	}
}

// Without an idle-connection timeout, connections parked on an orphaned
// transport are never reaped.
func TestUpdateProxySettingsSetsIdleConnTimeout(t *testing.T) {
	ex := newProxyTestExchange("http://127.0.0.1:3128")
	ex.UpdateProxySettings()

	tr, ok := ex.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", ex.httpClient.Transport)
	}
	if tr.IdleConnTimeout == 0 {
		t.Error("IdleConnTimeout is 0, so idle connections are never reaped; want a non-zero timeout")
	}
}

// Fetch can run concurrently, so the memoised path must be race-free.
func TestUpdateProxySettingsConcurrent(t *testing.T) {
	ex := newProxyTestExchange("http://127.0.0.1:3128")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ex.UpdateProxySettings()
		}()
	}
	wg.Wait()

	if ex.httpClient.Transport == nil {
		t.Error("Transport is nil after concurrent updates")
	}
}
