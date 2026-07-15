package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// CDPTab represents one entry from Chrome's /json endpoint.
type CDPTab struct {
	ID    string `json:"id"`
	Type  string `json:"type"` // "page", "background_page", "service_worker", …
	Title string `json:"title"`
	URL   string `json:"url"`
}

// CDPPoller polls the Chrome DevTools Protocol /json endpoint to enrich tab
// sessions with the full URL. It degrades gracefully when Chrome is not running
// with --remote-debugging-port.
type CDPPoller struct {
	mu           sync.Mutex
	client       *http.Client
	endpoint     string
	cache        []CDPTab
	enabled      bool      // false → no-op until next probe attempt
	lastProbe    time.Time // time of last connectivity probe
	probeBackoff time.Duration
	lastPoll     time.Time
	minInterval  time.Duration
}

// NewCDPPoller creates a CDPPoller targeting the given port.
// Call Probe() once at startup to check if the port is reachable.
func NewCDPPoller(port int) *CDPPoller {
	return &CDPPoller{
		client:       &http.Client{Timeout: time.Second},
		endpoint:     fmt.Sprintf("http://localhost:%d/json", port),
		minInterval:  500 * time.Millisecond,
		probeBackoff: 30 * time.Second,
	}
}

// Probe performs a one-shot connectivity check.
// Returns true if Chrome's debug port is reachable.
func (p *CDPPoller) Probe(ctx context.Context) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	ok := p.fetchAndCache(ctx)
	p.enabled = ok
	p.lastProbe = time.Now()
	return ok
}

// URLForTitle returns the URL of the Chrome tab whose title matches tabTitle.
// Returns "" when CDP is unavailable or no matching tab is found.
// The caller should invoke this only when Chrome/Edge is the active window.
func (p *CDPPoller) URLForTitle(ctx context.Context, tabTitle string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ctx.Err() != nil {
		return ""
	}

	if !p.enabled {
		// Retry probe after backoff.
		if time.Since(p.lastProbe) >= p.probeBackoff {
			p.enabled = p.fetchAndCache(ctx)
			p.lastProbe = time.Now()
		}
		if !p.enabled {
			return ""
		}
	}

	// Rate-limit fetches to minInterval.
	if time.Since(p.lastPoll) >= p.minInterval {
		_ = p.fetchAndCache(ctx) // non-fatal; stale cache is fine
		p.lastPoll = time.Now()
	}

	for _, tab := range p.cache {
		if tab.Type == "page" && tab.Title == tabTitle {
			return tab.URL
		}
	}
	return ""
}

// fetchAndCache performs the HTTP GET and updates p.cache. Must be called with p.mu held.
// Returns true on success.
func (p *CDPPoller) fetchAndCache(ctx context.Context) bool {
	if ctx.Err() != nil {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint, nil)
	if err != nil {
		return false
	}
	resp, err := p.client.Do(req)
	if err != nil {
		p.enabled = false
		return false
	}
	defer resp.Body.Close()
	var tabs []CDPTab
	if err := json.NewDecoder(resp.Body).Decode(&tabs); err != nil {
		return false
	}
	p.cache = tabs
	return true
}
