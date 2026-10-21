package pppoe

import (
	"errors"
	"fmt"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"github.com/go-routeros/routeros/v3"
	"gorm.io/gorm"
)

type Pool struct {
	Name   string `json:"name"`
	Ranges string `json:"ranges"`
}

func ListPools(userID uint, mikrotikID uint64) ([]Pool, error) {
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

	client, err := routeros.Dial(
		fmt.Sprintf("%s:%d", mikrotik.Host, mikrotik.Port),
		mikrotik.Username,
		password,
	)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(
		"/ip/pool/print",
		"=.proplist=name,ranges",
	)
	if err != nil {
		return nil, err
	}

	pools := make([]Pool, 0, len(reply.Re))

	for _, row := range reply.Re {
		pools = append(pools, Pool{
			Name:   row.Map["name"],
			Ranges: row.Map["ranges"],
		})
	}

	return pools, nil
}
