// Package mikrotikconn membuka sesi RouterOS yang dipakai bersama oleh service Hotspot.
package mikrotikconn

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"github.com/go-routeros/routeros/v3"
	"gorm.io/gorm"
)

const sessionTimeout = 30 * time.Second

var ErrMikrotikNotFound = errors.New("MikroTik not found")

// Connect memuat router milik user lalu membuka sesi API-nya.
func Connect(userID uint, mikrotikID uint64) (*routeros.Client, error) {
	mikrotik, password, err := load(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	return Dial(mikrotik, password)
}

// ConnectStream seperti Connect, tetapi sesinya tidak dibatasi deadline 30 detik sehingga bisa
// dipakai untuk perintah yang terus mengalir (mis. monitor-traffic). Pemanggil wajib menutup
// sesi dan menjaga sendiri batas waktunya.
func ConnectStream(userID uint, mikrotikID uint64) (*routeros.Client, error) {
	mikrotik, password, err := load(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	return dial(mikrotik, password, true)
}

func load(userID uint, mikrotikID uint64) (*models.Mikrotik, string, error) {
	var mikrotik models.Mikrotik

	err := config.DB.
		Where("id = ? AND user_id = ?", mikrotikID, userID).
		First(&mikrotik).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", ErrMikrotikNotFound
	}
	if err != nil {
		return nil, "", err
	}

	password, err := utils.DecryptMikrotikPassword(mikrotik.PasswordEncrypted)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decrypt MikroTik password: %w", err)
	}

	return &mikrotik, password, nil
}

// Dial membuka sesi RouterOS lewat dialer yang memblokir target internal (SSRF).
func Dial(mikrotik *models.Mikrotik, password string) (*routeros.Client, error) {
	return dial(mikrotik, password, false)
}

func dial(mikrotik *models.Mikrotik, password string, persistent bool) (*routeros.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), sessionTimeout)
	defer cancel()

	address := net.JoinHostPort(mikrotik.Host, strconv.Itoa(mikrotik.Port))

	conn, err := utils.SafeDialer(10*time.Second).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MikroTik: %w", err)
	}

	// Router yang menggantung tidak boleh menahan request tanpa batas.
	_ = conn.SetDeadline(time.Now().Add(sessionTimeout))

	client, err := routeros.NewClient(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := client.LoginContext(ctx, mikrotik.Username, password); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to log in to MikroTik: %w", err)
	}

	// Deadline hanya melindungi login; sesi stream dijaga pemanggilnya.
	if persistent {
		_ = conn.SetDeadline(time.Time{})
	}

	return client, nil
}
