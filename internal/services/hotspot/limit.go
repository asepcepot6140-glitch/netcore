package hotspot

import (
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// InputError adalah error validasi dengan pesan yang aman ditampilkan ke user.
type InputError struct{ Message string }

func (e *InputError) Error() string { return e.Message }

// Is membuat InputError cocok dengan ErrInvalidInput pada errors.Is.
func (e *InputError) Is(target error) bool { return target == ErrInvalidInput }

const (
	secondsPerMinute = 60
	secondsPerHour   = 60 * secondsPerMinute
	secondsPerDay    = 24 * secondsPerHour
	secondsPerWeek   = 7 * secondsPerDay
	// RouterOS tidak punya satuan bulan, jadi 1 bulan dihitung 30 hari.
	secondsPerMonth = 30 * secondsPerDay
)

var (
	uptimeSegment = regexp.MustCompile(`^(\d+)(mo|w|d|h|m|s)`)
	uptimeClock   = regexp.MustCompile(`^(\d+d )?\d{1,2}:\d{2}:\d{2}$`)
	bytesPattern  = regexp.MustCompile(`^(\d+(?:\.\d+)?)([kmgt]?)b?$`)

	uptimeUnits = map[string]uint64{
		"s":  1,
		"m":  secondsPerMinute,
		"h":  secondsPerHour,
		"d":  secondsPerDay,
		"w":  secondsPerWeek,
		"mo": secondsPerMonth,
	}
)

// NormalizeUptime mengubah input seperti "1h", "1H", "2d12h", "1mo" menjadi format RouterOS
// (w/d/h/m/s, huruf kecil). String kosong dipertahankan agar tetap berarti "hapus nilai".
func NormalizeUptime(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	if uptimeClock.MatchString(value) {
		return value, nil
	}

	rest := strings.Join(strings.Fields(value), "")
	if rest == "0" {
		return "0s", nil
	}

	var total uint64
	for rest != "" {
		match := uptimeSegment.FindStringSubmatch(rest)
		if match == nil {
			return "", &InputError{"Limit uptime tidak valid. Gunakan contoh: 30m, 1h, 1d, 1w, 1mo atau 1d12h"}
		}

		amount, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil || amount > math.MaxUint32 {
			return "", &InputError{"Limit uptime terlalu besar"}
		}

		total += amount * uptimeUnits[match[2]]
		if total > math.MaxUint32 {
			return "", &InputError{"Limit uptime terlalu besar"}
		}
		rest = rest[len(match[0]):]
	}

	return formatUptime(total), nil
}

func formatUptime(total uint64) string {
	if total == 0 {
		return "0s"
	}

	var out strings.Builder
	for _, part := range []struct {
		unit    string
		seconds uint64
	}{
		{"w", secondsPerWeek},
		{"d", secondsPerDay},
		{"h", secondsPerHour},
		{"m", secondsPerMinute},
		{"s", 1},
	} {
		if count := total / part.seconds; count > 0 {
			out.WriteString(strconv.FormatUint(count, 10) + part.unit)
			total %= part.seconds
		}
	}
	return out.String()
}

// NormalizeBytes mengubah input seperti "100M", "10g", "1.5GB" menjadi jumlah byte (basis 1024,
// sama seperti RouterOS). Angka tanpa satuan dianggap byte. String kosong dipertahankan.
func NormalizeBytes(value string) (string, error) {
	value = strings.ToLower(strings.Join(strings.Fields(value), ""))
	if value == "" {
		return "", nil
	}

	match := bytesPattern.FindStringSubmatch(value)
	if match == nil {
		return "", &InputError{"Limit bytes tidak valid. Gunakan contoh: 500M, 10G atau 1T"}
	}

	amount, ok := new(big.Rat).SetString(match[1])
	if !ok {
		return "", &InputError{"Limit bytes tidak valid"}
	}

	multiplier := int64(1)
	switch match[2] {
	case "k":
		multiplier = 1 << 10
	case "m":
		multiplier = 1 << 20
	case "g":
		multiplier = 1 << 30
	case "t":
		multiplier = 1 << 40
	}
	amount.Mul(amount, big.NewRat(multiplier, 1))

	// Pecahan byte dibuang (pembulatan ke bawah).
	bytes := new(big.Int).Quo(amount.Num(), amount.Denom())
	if !bytes.IsInt64() {
		return "", &InputError{"Limit bytes terlalu besar"}
	}
	return bytes.String(), nil
}

// NormalizeLimits menormalkan limit uptime dan bytes (in/out/total); nil (tidak diubah) dibiarkan.
func (input *UserInput) NormalizeLimits() error {
	if input.LimitUptime != nil {
		uptime, err := NormalizeUptime(*input.LimitUptime)
		if err != nil {
			return err
		}
		input.LimitUptime = &uptime
	}
	for _, value := range []**string{&input.LimitBytesIn, &input.LimitBytesOut, &input.LimitBytesTotal} {
		if *value == nil {
			continue
		}
		bytes, err := NormalizeBytes(**value)
		if err != nil {
			return err
		}
		*value = &bytes
	}
	return nil
}
