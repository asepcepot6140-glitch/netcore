package hotspot

import (
	"errors"
	"strconv"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/services/mikrotikconn"

	"github.com/go-routeros/routeros/v3"
)

var (
	ErrMikrotikNotFound = mikrotikconn.ErrMikrotikNotFound
	ErrUserNotFound     = errors.New("Hotspot User not found")
	ErrProfileNotFound  = errors.New("Hotspot Profile not found")
	ErrActiveNotFound   = errors.New("Hotspot Active session not found")
	ErrHostNotFound     = errors.New("Hotspot Host not found")
	ErrInvalidInput     = errors.New("invalid input")
	ErrUserExists       = errors.New("Hotspot User name already exists")
	ErrProfileExists    = errors.New("Hotspot Profile name already exists")
)

// MapDuplicate mengubah penolakan duplikat dari RouterOS menjadi duplicate;
// error lain dikembalikan apa adanya.
func MapDuplicate(err, duplicate error) error {
	var deviceErr *routeros.DeviceError
	if !errors.As(err, &deviceErr) || deviceErr.Sentence == nil {
		return err
	}

	message := strings.ToLower(deviceErr.Sentence.Map["message"])
	if strings.Contains(message, "already exists") || strings.Contains(message, "already have") {
		return duplicate
	}

	return err
}

func connect(userID uint, mikrotikID uint64) (*routeros.Client, error) {
	return mikrotikconn.Connect(userID, mikrotikID)
}

// Field memetakan satu properti opsional ke key RouterOS-nya.
type Field struct {
	key   string
	value *string
}

// CreateArgs hanya mengirim field yang terisi.
func CreateArgs(fields []Field) []string {
	var args []string
	for _, f := range fields {
		if v := optionalValue(f.value); v != "" {
			args = append(args, "="+f.key+"="+v)
		}
	}
	return args
}

// UpdateArgs mengirim setiap field yang tidak nil; string kosong menghapus nilai di router.
func UpdateArgs(fields []Field) []string {
	var args []string
	for _, f := range fields {
		if f.value != nil {
			args = append(args, "="+f.key+"="+strings.TrimSpace(*f.value))
		}
	}
	return args
}

func optionalValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
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

// itemExists memeriksa keberadaan item sebelum diubah atau dihapus.
func itemExists(client *routeros.Client, path, id string) (bool, error) {
	reply, err := client.Run(path+"/print", "?.id="+id, "=.proplist=.id")
	if err != nil {
		return false, err
	}
	return len(reply.Re) > 0, nil
}
