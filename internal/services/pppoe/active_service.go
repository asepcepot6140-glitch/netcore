package pppoe

import (
	"errors"
	"slices"

	"github.com/go-routeros/routeros/v3"
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
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return fetchActive(client)
}

const activeProperties = "=.proplist=.id,name,service,caller-id,address,uptime,encoding,session-id,limit-bytes-in,limit-bytes-out,radius"

// ActiveFromRow membaca satu baris /ppp/active/print.
func ActiveFromRow(row map[string]string) ActiveConnection {
	return ActiveConnection{
		ID:            row[".id"],
		Name:          row["name"],
		Service:       row["service"],
		CallerID:      row["caller-id"],
		Address:       row["address"],
		Uptime:        row["uptime"],
		Encoding:      row["encoding"],
		SessionID:     row["session-id"],
		LimitBytesIn:  row["limit-bytes-in"],
		LimitBytesOut: row["limit-bytes-out"],
		Radius:        row["radius"] == "true",
	}
}

// fetchActive membaca semua koneksi aktif lewat client yang sudah terhubung.
func fetchActive(client *routeros.Client) ([]ActiveConnection, error) {
	reply, err := client.Run("/ppp/active/print", activeProperties)
	if err != nil {
		return nil, err
	}

	connections := make([]ActiveConnection, 0, len(reply.Re))

	for _, row := range reply.Re {
		connections = append(connections, ActiveFromRow(row.Map))
	}

	// RouterOS mengembalikan data dari yang terlama; tampilkan yang terbaru lebih dulu.
	slices.Reverse(connections)

	return connections, nil
}

// RemoveActiveConnection memutus koneksi PPP aktif berdasarkan .id RouterOS.
func RemoveActiveConnection(userID uint, mikrotikID uint64, id string) error {
	if id == "" {
		return ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
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
