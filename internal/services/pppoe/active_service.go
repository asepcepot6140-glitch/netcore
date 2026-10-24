package pppoe

import (
	"errors"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"gorm.io/gorm"
)

var ErrActiveConnectionNotFound = errors.New("active PPP connection not found")

type ActiveConnection struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Service       string `json:"service"`
	CallerID      string `json:"caller_id"`
	Address       string `json:"address"`
	Uptime        string `json:"uptime"`
	Encoding      string `json:"encoding"`
	SessionID     string `json:"session_id"`
	LimitBytesIn  string `json:"limit_bytes_in"`
	LimitBytesOut string `json:"limit_bytes_out"`
	Radius        bool   `json:"radius"`
}

func ListActiveConnections(userID uint, mikrotikID uint64) ([]ActiveConnection, error) {
	var mikrotik models.Mikrotik

	err := config.DB.
		Where("id = ? AND user_id = ?", mikrotikID, userID).
		First(&mikrotik).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMikrotikNotFound
	}
	if err != nil {
		return nil, err
	}

	password, err := utils.DecryptMikrotikPassword(mikrotik.PasswordEncrypted)
	if err != nil {
		return nil, err
	}

	client, err := dialMikrotik(&mikrotik, password)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(
		"/ppp/active/print",
		"=.proplist=.id,name,service,caller-id,address,uptime,encoding,session-id,limit-bytes-in,limit-bytes-out,radius",
	)
	if err != nil {
		return nil, err
	}

	connections := make([]ActiveConnection, 0, len(reply.Re))

	for _, row := range reply.Re {
		connections = append(connections, ActiveConnection{
			ID:            row.Map[".id"],
			Name:          row.Map["name"],
			Service:       row.Map["service"],
			CallerID:      row.Map["caller-id"],
			Address:       row.Map["address"],
			Uptime:        row.Map["uptime"],
			Encoding:      row.Map["encoding"],
			SessionID:     row.Map["session-id"],
			LimitBytesIn:  row.Map["limit-bytes-in"],
			LimitBytesOut: row.Map["limit-bytes-out"],
			Radius:        row.Map["radius"] == "true",
		})
	}

	return connections, nil
}

// RemoveActiveConnection memutus koneksi PPP aktif berdasarkan .id RouterOS.
func RemoveActiveConnection(userID uint, mikrotikID uint64, id string) error {
	if id == "" {
		return ErrInvalidInput
	}

	var mikrotik models.Mikrotik

	err := config.DB.
		Where("id = ? AND user_id = ?", mikrotikID, userID).
		First(&mikrotik).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrMikrotikNotFound
	}
	if err != nil {
		return err
	}

	password, err := utils.DecryptMikrotikPassword(mikrotik.PasswordEncrypted)
	if err != nil {
		return err
	}

	client, err := dialMikrotik(&mikrotik, password)
	if err != nil {
		return err
	}
	defer client.Close()

	// Pastikan koneksi masih aktif dan ambil .id RouterOS yang valid.
	reply, err := client.Run(
		"/ppp/active/print",
		"?.id="+id,
		"=.proplist=.id",
	)
	if err != nil {
		return err
	}

	if len(reply.Re) == 0 {
		return ErrActiveConnectionNotFound
	}

	_, err = client.Run(
		"/ppp/active/remove",
		"=.id="+reply.Re[0].Map[".id"],
	)

	return err
}
