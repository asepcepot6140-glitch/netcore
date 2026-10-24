package utils

import (
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"
)

var ErrBlockedTarget = errors.New("target address is not allowed")

// targetAllowed memeriksa IP terhadap TARGET_ALLOWLIST (daftar IP/CIDR, dipisah koma).
func targetAllowed(ip net.IP) bool {
	for _, entry := range strings.Split(os.Getenv("TARGET_ALLOWLIST"), ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if _, cidr, err := net.ParseCIDR(entry); err == nil {
			if cidr.Contains(ip) {
				return true
			}
			continue
		}

		if allowed := net.ParseIP(entry); allowed != nil && allowed.Equal(ip) {
			return true
		}
	}

	return false
}

// CheckTargetIP menolak loopback, link-local (termasuk metadata cloud), dan multicast.
// Range privat RFC1918 tetap diizinkan karena perangkat jaringan umumnya berada di sana.
func CheckTargetIP(ip net.IP) error {
	blocked := ip.IsLoopback() ||
		ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsMulticast() ||
		ip.Equal(net.IPv4bcast)

	if blocked && !targetAllowed(ip) {
		return ErrBlockedTarget
	}

	return nil
}

// SafeDialer memvalidasi IP hasil resolve sebelum koneksi dibuat, sehingga DNS rebinding tidak lolos.
func SafeDialer(timeout time.Duration) *net.Dialer {
	return &net.Dialer{
		Timeout: timeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return ErrBlockedTarget
			}

			ip := net.ParseIP(host)
			if ip == nil {
				return ErrBlockedTarget
			}

			return CheckTargetIP(ip)
		},
	}
}

// NewSafeHTTPClient tidak memakai proxy dan tidak mengikuti redirect agar guard dial tidak terlewati.
func NewSafeHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:         SafeDialer(10 * time.Second).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			MaxIdleConns:        20,
			IdleConnTimeout:     60 * time.Second,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
