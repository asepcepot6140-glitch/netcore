package simplequeue

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/asepcepot6140-glitch/netcore/internal/services/mikrotikconn"

	"github.com/go-routeros/routeros/v3"
)

var (
	ErrMikrotikNotFound = mikrotikconn.ErrMikrotikNotFound
	ErrNotFound         = errors.New("Simple Queue not found")
	ErrExists           = errors.New("Simple Queue name already exists")
	ErrDynamic          = errors.New("Dynamic Simple Queue cannot be modified")
	ErrInvalidInput     = errors.New("invalid input")
)

// InputError adalah error validasi dengan pesan yang aman ditampilkan ke user.
type InputError struct{ Message string }

func (e *InputError) Error() string { return e.Message }

// Is membuat InputError cocok dengan ErrInvalidInput pada errors.Is.
func (e *InputError) Is(target error) bool { return target == ErrInvalidInput }

const maxDeviceMessageLength = 200

func connect(userID uint, mikrotikID uint64) (*routeros.Client, error) {
	return mikrotikconn.Connect(userID, mikrotikID)
}

// mapRouterError mengubah penolakan perintah oleh RouterOS (!trap) menjadi error yang bisa
// ditampilkan: duplikat jadi ErrExists, alasan lain (mis. burst tidak konsisten) diteruskan sebagai
// InputError. Error koneksi dikembalikan apa adanya.
func mapRouterError(err error) error {
	var deviceErr *routeros.DeviceError
	if err == nil || !errors.As(err, &deviceErr) || deviceErr.Sentence == nil {
		return err
	}

	message := strings.TrimSpace(deviceErr.Sentence.Map["message"])
	lower := strings.ToLower(message)
	if strings.Contains(lower, "already exists") || strings.Contains(lower, "already have") {
		return ErrExists
	}
	if message == "" {
		return err
	}

	if utf8.RuneCountInString(message) > maxDeviceMessageLength {
		message = string([]rune(message)[:maxDeviceMessageLength])
	}
	return &InputError{Message: "MikroTik: " + message}
}

// ValidID memastikan ID berformat RouterOS (*1A) sebelum dipakai di query.
func ValidID(id string) bool {
	return strings.HasPrefix(id, "*") && len(id) > 1
}

func ParseMikrotikID(value string) (uint64, error) {
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, ErrInvalidInput
	}
	return id, nil
}
