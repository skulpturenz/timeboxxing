package platform

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// publicIPEndpoint is an external echo service returning the caller's public IP
// as plain text. The public IP of a NAT'd machine cannot be discovered locally,
// so an outbound request is required (see the plan's egress caveat).
const publicIPEndpoint = "https://api.ipify.org"

// publicIPRefreshInterval is how often the cached public IP is refreshed. Public
// IPs change rarely, so a long interval keeps egress negligible.
const publicIPRefreshInterval = 15 * time.Minute

// publicIPProvider caches the machine's public IP, refreshed by a background
// goroutine. Reads are non-blocking so the poll loop is never stalled on network
// I/O; the value is nil until the first successful lookup.
type publicIPProvider struct {
	mu    sync.RWMutex
	value *string

	client *http.Client
	start  sync.Once
}

var defaultPublicIPProvider = &publicIPProvider{
	client: &http.Client{Timeout: 5 * time.Second},
}

// currentPublicIP returns the cached public IP (a fresh copy so callers cannot
// mutate the cache), lazily starting the background refresher on first use.
// Returns nil until a lookup has succeeded.
func currentPublicIP() *string {
	defaultPublicIPProvider.ensureStarted()

	defaultPublicIPProvider.mu.RLock()
	defer defaultPublicIPProvider.mu.RUnlock()
	if defaultPublicIPProvider.value == nil {
		return nil
	}
	ip := *defaultPublicIPProvider.value
	return &ip
}

func (p *publicIPProvider) ensureStarted() {
	p.start.Do(func() {
		go p.run(context.Background())
	})
}

func (p *publicIPProvider) run(ctx context.Context) {
	// Fetch once immediately, then on a long interval.
	p.refresh(ctx)

	ticker := time.NewTicker(publicIPRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.refresh(ctx)
		}
	}
}

// refresh performs one lookup. On failure it leaves the previous cached value
// untouched (a transient network error should not blank out a known IP).
func (p *publicIPProvider) refresh(ctx context.Context) {
	ip, err := p.lookup(ctx)
	if err != nil {
		slog.Debug("public ip lookup failed", "error", err)
		return
	}

	p.mu.Lock()
	p.value = &ip
	p.mu.Unlock()
}

func (p *publicIPProvider) lookup(ctx context.Context) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, p.client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, publicIPEndpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}

	ip := parsePublicIP(string(body))
	if ip == "" {
		return "", &net.AddrError{Err: "invalid public ip response", Addr: strings.TrimSpace(string(body))}
	}
	return ip, nil
}

// parsePublicIP trims and validates an echo-service response body, returning the
// IP string or "" if it is not a valid IP.
func parsePublicIP(body string) string {
	ip := strings.TrimSpace(body)
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}
