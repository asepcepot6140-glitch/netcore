package hotspot

import (
	"cmp"
	"slices"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"github.com/go-routeros/routeros/v3"
)

const activePath = "/ip/hotspot/active"

const activeProperties = ".id,user,address,mac-address,server,login-by,uptime,idle-time,session-time-left,bytes-in,bytes-out"

// Active adalah sesi hotspot yang sedang online (read-only, bisa di-kick).
type Active struct {
	ID              string `json:"id"`
	User            string `json:"user"`
	Address         string `json:"address"`
	MacAddress      string `json:"mac_address"`
	Server          string `json:"server"`
	LoginBy         string `json:"login_by"`
	Uptime          string `json:"uptime"`
	IdleTime        string `json:"idle_time"`
	SessionTimeLeft string `json:"session_time_left"`
	BytesIn         string `json:"bytes_in"`
	BytesOut        string `json:"bytes_out"`
}

func ActiveFromRow(row map[string]string) Active {
	return Active{
		ID:              row[".id"],
		User:            row["user"],
		Address:         row["address"],
		MacAddress:      row["mac-address"],
		Server:          row["server"],
		LoginBy:         row["login-by"],
		Uptime:          row["uptime"],
		IdleTime:        row["idle-time"],
		SessionTimeLeft: row["session-time-left"],
		BytesIn:         row["bytes-in"],
		BytesOut:        row["bytes-out"],
	}
}

func ListActive(userID uint, mikrotikID uint64) ([]Active, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return fetchActive(client)
}

// fetchActive membaca semua sesi aktif lewat client yang sudah terhubung.
func fetchActive(client *routeros.Client) ([]Active, error) {
	reply, err := client.Run(activePath+"/print", "=.proplist="+activeProperties)
	if err != nil {
		return nil, err
	}

	sessions := make([]Active, 0, len(reply.Re))
	for _, row := range reply.Re {
		sessions = append(sessions, ActiveFromRow(row.Map))
	}

	OrderActiveNewestFirst(sessions)

	return sessions, nil
}

// KickActive memutus sesi hotspot; user harus login ulang.
func KickActive(userID uint, mikrotikID uint64, id string) error {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	exists, err := itemExists(client, activePath, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrActiveNotFound
	}

	_, err = client.Run(activePath+"/remove", "=.id="+id)
	return err
}

// OrderActiveNewestFirst mengurutkan sesi dengan uptime terpendek (paling baru login) di atas.
// Sesi tanpa uptime yang valid ditaruh paling bawah; urutan sama dipertahankan dari print RouterOS.
func OrderActiveNewestFirst(sessions []Active) {
	slices.Reverse(sessions)
	slices.SortStableFunc(sessions, func(a, b Active) int {
		first, firstOK := utils.ParseRouterOSDuration(a.Uptime)
		second, secondOK := utils.ParseRouterOSDuration(b.Uptime)

		switch {
		case firstOK && secondOK:
			return cmp.Compare(first, second)
		case firstOK:
			return -1
		case secondOK:
			return 1
		}

		return 0
	})
}
