package genieacs

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	schemeProbeTimeout = 3 * time.Second
	schemeFoundTTL     = 5 * time.Minute
	// Hasil gagal dicache singkat agar host mati tidak menahan setiap request (polling).
	schemeFailedTTL = 15 * time.Second
)

type schemeEntry struct {
	scheme  string
	expires time.Time
}

var schemeCache sync.Map // "host:port" -> schemeEntry

// WithScheme melengkapi host tanpa skema (mis. "192.168.1.10") dengan http atau https
// berdasarkan hasil probe, sehingga user cukup mengisi host saja.
func WithScheme(host string, port int) string {
	if strings.Contains(host, "://") {
		return host
	}

	key := host + "|" + strconv.Itoa(port)
	if cached, ok := schemeCache.Load(key); ok {
		entry := cached.(schemeEntry)
		if time.Now().Before(entry.expires) {
			return entry.scheme + "://" + host
		}
	}

	scheme, found := probeScheme(host, port)
	ttl := schemeFailedTTL
	if found {
		ttl = schemeFoundTTL
	}
	schemeCache.Store(key, schemeEntry{scheme: scheme, expires: time.Now().Add(ttl)})

	return scheme + "://" + host
}

// probeScheme mencoba http lalu https. Jika keduanya gagal, fallback ke http agar
// error koneksi asli muncul dari request berikutnya.
func probeScheme(host string, port int) (string, bool) {
	for _, scheme := range []string{"http", "https"} {
		target, err := url.Parse(scheme + "://" + host)
		if err != nil || target.Hostname() == "" {
			return "http", false
		}

		if target.Port() == "" {
			if port <= 0 || port > 65535 {
				return "http", false
			}
			target.Host = net.JoinHostPort(target.Hostname(), strconv.Itoa(port))
		}

		ctx, cancel := context.WithTimeout(context.Background(), schemeProbeTimeout)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
		if err != nil {
			cancel()
			return "http", false
		}

		resp, err := httpClient.Do(req)
		cancel()
		if err != nil {
			continue
		}
		_ = resp.Body.Close()

		// Server TLS membalas 400 saat dihubungi lewat http polos.
		if scheme == "http" && resp.StatusCode == http.StatusBadRequest {
			continue
		}

		return scheme, true
	}

	return "http", false
}
