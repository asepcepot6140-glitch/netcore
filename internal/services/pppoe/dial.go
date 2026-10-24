package pppoe

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"github.com/go-routeros/routeros/v3"
)

const mikrotikSessionTimeout = 30 * time.Second

// dialMikrotik membuka sesi RouterOS lewat dialer yang memblokir target internal (SSRF).
func dialMikrotik(mikrotik *models.Mikrotik, password string) (*routeros.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mikrotikSessionTimeout)
	defer cancel()

	address := net.JoinHostPort(mikrotik.Host, strconv.Itoa(mikrotik.Port))

	conn, err := utils.SafeDialer(10*time.Second).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MikroTik: %w", err)
	}

	// Router yang menggantung tidak boleh menahan request tanpa batas.
	_ = conn.SetDeadline(time.Now().Add(mikrotikSessionTimeout))

	client, err := routeros.NewClient(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := client.LoginContext(ctx, mikrotik.Username, password); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to log in to MikroTik: %w", err)
	}

	return client, nil
}
