package hotspot

import (
	"slices"
	"strings"

	"github.com/go-routeros/routeros/v3"
)

const hostPath = "/ip/hotspot/host"

const hostProperties = ".id,mac-address,address,to-address,server,uptime,idle-time,bytes-in,bytes-out,authorized,bypassed,DHCP,dynamic,static,comment"

// Host adalah perangkat yang terdeteksi hotspot, login maupun belum.
type Host struct {
	ID         string `json:"id"`
	MacAddress string `json:"mac_address"`
	Address    string `json:"address"`
	ToAddress  string `json:"to_address"`
	Server     string `json:"server"`
	Uptime     string `json:"uptime"`
	IdleTime   string `json:"idle_time"`
	BytesIn    string `json:"bytes_in"`
	BytesOut   string `json:"bytes_out"`
	Authorized bool   `json:"authorized"`
	Bypassed   bool   `json:"bypassed"`
	DHCP       bool   `json:"dhcp"`
	Dynamic    bool   `json:"dynamic"`
	Comment    string `json:"comment"`
}

func HostFromRow(row map[string]string) Host {
	dhcp := row["DHCP"] == "true"
	// RouterOS tidak selalu mengirim "dynamic": host yang bukan DHCP dan bukan static berflag D di Winbox.
	dynamic := !dhcp && row["static"] != "true"
	if value, ok := row["dynamic"]; ok {
		dynamic = value == "true"
	}

	return Host{
		ID:         row[".id"],
		MacAddress: row["mac-address"],
		Address:    row["address"],
		ToAddress:  row["to-address"],
		Server:     row["server"],
		Uptime:     row["uptime"],
		IdleTime:   row["idle-time"],
		BytesIn:    row["bytes-in"],
		BytesOut:   row["bytes-out"],
		Authorized: row["authorized"] == "true",
		Bypassed:   row["bypassed"] == "true",
		DHCP:       dhcp,
		Dynamic:    dynamic,
		Comment:    row["comment"],
	}
}

func ListHosts(userID uint, mikrotikID uint64) ([]Host, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return fetchHosts(client)
}

// fetchHosts membaca semua host lewat client yang sudah terhubung.
func fetchHosts(client *routeros.Client) ([]Host, error) {
	reply, err := client.Run(hostPath+"/print", "=.proplist="+hostProperties)
	if err != nil {
		return nil, err
	}

	hosts := make([]Host, 0, len(reply.Re))
	for _, row := range reply.Re {
		hosts = append(hosts, HostFromRow(row.Map))
	}

	// RouterOS mengembalikan data dari yang terlama; tampilkan yang terbaru lebih dulu.
	slices.Reverse(hosts)

	return hosts, nil
}

// DeleteHost menghapus entri host; host dinamis akan terdeteksi ulang bila perangkat masih aktif.
func DeleteHost(userID uint, mikrotikID uint64, id string) error {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	exists, err := itemExists(client, hostPath, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrHostNotFound
	}

	_, err = client.Run(hostPath+"/remove", "=.id="+id)
	return err
}
