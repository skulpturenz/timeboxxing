package location

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/memo"
)

// The public IP of a NAT'd machine cannot be discovered locally, so an outbound
// request to an echo service is required. Public IPs change rarely, so the memo
// entry lives a long time — the lookup runs at most once per TTL.
const (
	defaultPublicIPEndpoint = "https://api.ipify.org"
	publicIPCacheTTL        = time.Hour
	publicIPKey             = "public_ip"
)

// IPProvider resolves the machine's public IP and memoizes it in a memo (TTL
// cache + singleflight). PublicIP performs the HTTP lookup inline on a cache
// miss — it may block up to the client timeout — and returns the cached value on
// subsequent calls until the TTL expires. Satisfies PublicIPProvider.
type IPProvider struct {
	endpoint string
	client   *http.Client
	cache    *memo.Memoize
}

// NewPublicIPProvider returns a provider using the default echo endpoint.
func NewPublicIPProvider() *IPProvider {
	ttl := publicIPCacheTTL
	cleanup := publicIPCacheTTL + time.Minute
	return &IPProvider{
		endpoint: defaultPublicIPEndpoint,
		client:   &http.Client{Timeout: 5 * time.Second},
		cache:    memo.NewWithOptions(&ttl, &cleanup),
	}
}

// PublicIP returns the machine's public IP (a fresh copy so callers cannot mutate
// the cache). The lookup is memoized: the first call (and the first after the TTL
// expires) performs the HTTP request and may block; other calls return the cached
// value. Returns nil when no lookup has succeeded (memo caches only successes).
func (p *IPProvider) PublicIP() *string {
	value, _, _ := p.cache.Do(publicIPKey, func() (any, error) {
		return p.lookup(context.Background())
	})

	ip, ok := value.(string)
	if !ok || ip == "" {
		return nil
	}
	return new(ip)
}

func (p *IPProvider) lookup(ctx context.Context) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, p.client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, p.endpoint, nil)
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
