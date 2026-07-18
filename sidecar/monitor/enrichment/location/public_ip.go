package location

import (
	"context"
	"net"
	"strings"
	"time"

	"resty.dev/v3"

	"github.com/skulpturenz/timeboxxing/sidecar/memo"
)

// publicIPTimeout bounds a single lookup attempt; publicIPBodyLimit caps the echo
// response (an IP string is tiny) so a misbehaving endpoint can't stream unbounded.
const (
	publicIPTimeout   = 5 * time.Second
	publicIPBodyLimit = 64
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
	client   *resty.Client
	cache    *memo.Memoize
}

// NewPublicIPProvider returns a provider using the default echo endpoint. The
// resty client retries transient failures (transport errors, 429, 5xx) with
// backoff; a 4xx is not retried.
func NewPublicIPProvider() *IPProvider {
	ttl := publicIPCacheTTL
	cleanup := publicIPCacheTTL + time.Minute
	client := resty.New().
		SetTimeout(publicIPTimeout).
		SetResponseBodyLimit(publicIPBodyLimit).
		SetRetryCount(2).
		SetRetryWaitTime(200*time.Millisecond).
		SetRetryMaxWaitTime(2*time.Second).
		SetRetryDefaultConditions(true).
		AddRetryConditions(resty.RetryConditionStatusTooManyRequests, resty.RetryConditionStatus5XX)
	return &IPProvider{
		endpoint: defaultPublicIPEndpoint,
		client:   client,
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
	// resty applies the per-attempt timeout (SetTimeout) and body limit; ctx keeps
	// cancellation propagating across retries.
	resp, err := p.client.R().SetContext(ctx).Get(p.endpoint)
	if err != nil {
		return "", err
	}

	body := resp.Bytes()
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
