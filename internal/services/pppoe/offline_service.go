package pppoe

import (
	"errors"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"gorm.io/gorm"
)

type OfflineConnection struct {
	ID                   string `json:"id"`
	Disabled             bool   `json:"disabled"`
	Name                 string `json:"name"`
	Service              string `json:"service"`
	CallerID             string `json:"caller_id"`
	Password             string `json:"password"`
	Profile              string `json:"profile"`
	RemoteAddress        string `json:"remote_address"`
	LocalAddress         string `json:"local_address"`
	LastLoggedOut        string `json:"last_logged_out"`
	LastCallerID         string `json:"last_caller_id"`
	LastDisconnectReason string `json:"last_disconnect_reason"`
	Comment              string `json:"comment"`
}

func ListOfflineConnections(userID uint, mikrotikID uint64) ([]OfflineConnection, error) {
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

	// Ambil seluruh PPP Secret.
	secretsReply, err := client.Run(
		"/ppp/secret/print",
		"=.proplist=.id,disabled,name,service,caller-id,password,profile,remote-address,local-address,last-logged-out,last-caller-id,last-disconnect-reason,comment",
	)
	if err != nil {
		return nil, err
	}

	// Ambil username yang sedang aktif.
	activeReply, err := client.Run(
		"/ppp/active/print",
		"=.proplist=name",
	)
	if err != nil {
		return nil, err
	}

	activeUsers := make(map[string]bool, len(activeReply.Re))

	for _, row := range activeReply.Re {
		activeUsers[row.Map["name"]] = true
	}

	offline := make([]OfflineConnection, 0)

	for _, row := range secretsReply.Re {
		name := row.Map["name"]

		// Lewati Secret yang sedang aktif.
		if activeUsers[name] {
			continue
		}

		offline = append(offline, OfflineConnection{
			ID:                   row.Map[".id"],
			Disabled:             row.Map["disabled"] == "true",
			Name:                 name,
			Service:              row.Map["service"],
			CallerID:             row.Map["caller-id"],
			Password:             row.Map["password"],
			Profile:              row.Map["profile"],
			RemoteAddress:        row.Map["remote-address"],
			LocalAddress:         row.Map["local-address"],
			LastLoggedOut:        row.Map["last-logged-out"],
			LastCallerID:         row.Map["last-caller-id"],
			LastDisconnectReason: row.Map["last-disconnect-reason"],
			Comment:              row.Map["comment"],
		})
	}

	return offline, nil
}
