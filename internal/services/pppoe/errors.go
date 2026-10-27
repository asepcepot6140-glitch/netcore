package pppoe

import (
	"errors"
	"strings"

	"github.com/go-routeros/routeros/v3"
)

// ErrNameExists berarti RouterOS menolak karena nama sudah dipakai item lain.
var ErrNameExists = errors.New("name already exists")

// MapDeviceError mengubah penolakan duplikat dari RouterOS menjadi ErrNameExists;
// error lain dikembalikan apa adanya.
func MapDeviceError(err error) error {
	var deviceErr *routeros.DeviceError
	if !errors.As(err, &deviceErr) || deviceErr.Sentence == nil {
		return err
	}

	message := strings.ToLower(deviceErr.Sentence.Map["message"])
	if strings.Contains(message, "already exists") || strings.Contains(message, "already have") {
		return ErrNameExists
	}

	return err
}
