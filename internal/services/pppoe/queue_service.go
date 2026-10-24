package pppoe

import (
	"errors"
	"sort"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"gorm.io/gorm"
)

func ListQueueNames(userID uint, mikrotikID uint64) ([]string, error) {
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

	queueNames := make(map[string]struct{})
	for _, command := range []string{"/queue/simple/print", "/queue/tree/print"} {
		reply, err := client.Run(command, "=.proplist=name")
		if err != nil {
			return nil, err
		}
		for _, row := range reply.Re {
			if name := row.Map["name"]; name != "" {
				queueNames[name] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(queueNames))
	for name := range queueNames {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}
