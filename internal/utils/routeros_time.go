package utils

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var routerOSDurationPart = regexp.MustCompile(`(\d+)(w|d|h|ms|m|s)`)

var routerOSDurationUnit = map[string]time.Duration{
	"w":  7 * 24 * time.Hour,
	"d":  24 * time.Hour,
	"h":  time.Hour,
	"m":  time.Minute,
	"s":  time.Second,
	"ms": time.Millisecond,
}

// ParseRouterOSDuration membaca durasi RouterOS seperti "1w2d3h4m5s" atau "45s".
// ok bernilai false jika kosong atau formatnya tidak dikenali.
func ParseRouterOSDuration(value string) (d time.Duration, ok bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	parts := routerOSDurationPart.FindAllStringSubmatch(value, -1)

	consumed := 0
	for _, part := range parts {
		consumed += len(part[0])

		n, err := strconv.ParseInt(part[1], 10, 64)
		if err != nil {
			return 0, false
		}

		d += time.Duration(n) * routerOSDurationUnit[part[2]]
	}

	// Seluruh teks harus berupa pasangan angka+satuan, bukan sekadar mengandung salah satunya.
	if len(parts) == 0 || consumed != len(value) {
		return 0, false
	}

	return d, true
}

// ParseRouterOSTime membaca waktu RouterOS seperti "oct/26/2026 10:26:48" (jam boleh kosong).
// Zona waktu router tidak diketahui, jadi hasilnya hanya layak untuk membandingkan urutan.
func ParseRouterOSTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)

	for _, layout := range []string{"Jan/2/2006 15:04:05", "Jan/2/2006"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}

	return time.Time{}, false
}
