package hotspot

import (
	"slices"
	"strconv"
	"strings"

	"github.com/go-routeros/routeros/v3"
)

const userPath = "/ip/hotspot/user"

const userProperties = ".id,name,password,profile,server,address,mac-address,limit-uptime,limit-bytes-in,limit-bytes-out,limit-bytes-total,uptime,bytes-in,bytes-out,comment,disabled"

type User struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Password        string `json:"password"`
	Profile         string `json:"profile"`
	Server          string `json:"server"`
	Address         string `json:"address"`
	MacAddress      string `json:"mac_address"`
	LimitUptime     string `json:"limit_uptime"`
	LimitBytesIn    string `json:"limit_bytes_in"`
	LimitBytesOut   string `json:"limit_bytes_out"`
	LimitBytesTotal string `json:"limit_bytes_total"`
	Uptime          string `json:"uptime"`
	BytesIn         string `json:"bytes_in"`
	BytesOut        string `json:"bytes_out"`
	Comment         string `json:"comment"`
	Disabled        bool   `json:"disabled"`
}

// UserInput: saat update, nil = tidak diubah dan "" = hapus nilai; saat create, nilai kosong dilewati.
type UserInput struct {
	Name            string  `json:"name"`
	Password        *string `json:"password"`
	Profile         *string `json:"profile"`
	Server          *string `json:"server"`
	Address         *string `json:"address"`
	MacAddress      *string `json:"mac_address"`
	LimitUptime     *string `json:"limit_uptime"`
	LimitBytesIn    *string `json:"limit_bytes_in"`
	LimitBytesOut   *string `json:"limit_bytes_out"`
	LimitBytesTotal *string `json:"limit_bytes_total"`
	Comment         *string `json:"comment"`
	Disabled        *bool   `json:"disabled"`
}

// Fields memuat semua properti opsional selain name dan disabled.
func (input UserInput) Fields() []Field {
	return []Field{
		{"password", input.Password},
		{"profile", input.Profile},
		{"server", input.Server},
		{"address", input.Address},
		{"mac-address", input.MacAddress},
		{"limit-uptime", input.LimitUptime},
		{"limit-bytes-in", input.LimitBytesIn},
		{"limit-bytes-out", input.LimitBytesOut},
		{"limit-bytes-total", input.LimitBytesTotal},
		{"comment", input.Comment},
	}
}

func UserFromRow(row map[string]string) User {
	// RouterOS tidak mengirim "server" bila nilainya default; kosong berarti semua server.
	server := row["server"]
	if server == "" {
		server = defaultServer
	}

	return User{
		ID:              row[".id"],
		Name:            row["name"],
		Password:        row["password"],
		Profile:         row["profile"],
		Server:          server,
		Address:         row["address"],
		MacAddress:      row["mac-address"],
		LimitUptime:     row["limit-uptime"],
		LimitBytesIn:    row["limit-bytes-in"],
		LimitBytesOut:   row["limit-bytes-out"],
		LimitBytesTotal: row["limit-bytes-total"],
		Uptime:          row["uptime"],
		BytesIn:         row["bytes-in"],
		BytesOut:        row["bytes-out"],
		Comment:         row["comment"],
		Disabled:        row["disabled"] == "true",
	}
}

func ListUsers(userID uint, mikrotikID uint64) ([]User, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return fetchUsers(client)
}

// fetchUsers membaca semua user lewat client yang sudah terhubung.
func fetchUsers(client *routeros.Client) ([]User, error) {
	reply, err := client.Run(userPath+"/print", "=.proplist="+userProperties)
	if err != nil {
		return nil, err
	}

	users := make([]User, 0, len(reply.Re))
	for _, row := range reply.Re {
		users = append(users, UserFromRow(row.Map))
	}

	// RouterOS mengembalikan data dari yang terlama; tampilkan yang terbaru lebih dulu.
	slices.Reverse(users)

	return users, nil
}

func ShowUser(userID uint, mikrotikID uint64, id string) (*User, error) {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return nil, ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(userPath+"/print", "?.id="+id, "=.proplist="+userProperties)
	if err != nil {
		return nil, err
	}
	if len(reply.Re) == 0 {
		return nil, ErrUserNotFound
	}

	user := UserFromRow(reply.Re[0].Map)
	return &user, nil
}

func CreateUser(userID uint, mikrotikID uint64, input UserInput) error {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return ErrInvalidInput
	}
	if err := input.NormalizeLimits(); err != nil {
		return err
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	args := append([]string{userPath + "/add", "=name=" + input.Name}, CreateArgs(input.Fields())...)
	if input.Disabled != nil && *input.Disabled {
		args = append(args, "=disabled=yes")
	}

	_, err = client.Run(args...)
	return MapDuplicate(err, ErrUserExists)
}

func UpdateUser(userID uint, mikrotikID uint64, id string, input UserInput) error {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return ErrInvalidInput
	}
	if err := input.NormalizeLimits(); err != nil {
		return err
	}

	args := []string{userPath + "/set", "=.id=" + id}

	if name := strings.TrimSpace(input.Name); name != "" {
		args = append(args, "=name="+name)
	}
	// Password kosong berarti tidak diubah; hotspot user tidak boleh tanpa password lewat form edit.
	fields := input.Fields()
	if input.Password != nil && strings.TrimSpace(*input.Password) == "" {
		fields[0].value = nil
	}
	args = append(args, UpdateArgs(fields)...)
	if input.Disabled != nil {
		args = append(args, "=disabled="+strconv.FormatBool(*input.Disabled))
	}

	if len(args) == 2 {
		return ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	exists, err := itemExists(client, userPath, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrUserNotFound
	}

	_, err = client.Run(args...)
	return MapDuplicate(err, ErrUserExists)
}

func DeleteUser(userID uint, mikrotikID uint64, id string) error {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	exists, err := itemExists(client, userPath, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrUserNotFound
	}

	_, err = client.Run(userPath+"/remove", "=.id="+id)
	return err
}
