package hotspot

import (
	"slices"
	"strings"
)

const profilePath = "/ip/hotspot/user/profile"

const profileProperties = ".id,name,address-pool,shared-users,rate-limit,session-timeout,idle-timeout,keepalive-timeout,status-autorefresh,add-mac-cookie,mac-cookie-timeout,address-list,insert-queue-before,parent-queue,queue-type,transparent-proxy,on-login,on-logout,comment"

type Profile struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	AddressPool       string `json:"address_pool"`
	SharedUsers       string `json:"shared_users"`
	RateLimit         string `json:"rate_limit"`
	SessionTimeout    string `json:"session_timeout"`
	IdleTimeout       string `json:"idle_timeout"`
	KeepaliveTimeout  string `json:"keepalive_timeout"`
	StatusAutorefresh string `json:"status_autorefresh"`
	AddMacCookie      string `json:"add_mac_cookie"`
	MacCookieTimeout  string `json:"mac_cookie_timeout"`
	AddressList       string `json:"address_list"`
	InsertQueueBefore string `json:"insert_queue_before"`
	ParentQueue       string `json:"parent_queue"`
	QueueType         string `json:"queue_type"`
	TransparentProxy  string `json:"transparent_proxy"`
	OnLogin           string `json:"on_login"`
	OnLogout          string `json:"on_logout"`
	Comment           string `json:"comment"`
}

// ProfileInput: saat update, nil = tidak diubah dan "" = hapus nilai; saat create, nilai kosong dilewati.
type ProfileInput struct {
	Name              string  `json:"name"`
	AddressPool       *string `json:"address_pool"`
	SharedUsers       *string `json:"shared_users"`
	RateLimit         *string `json:"rate_limit"`
	SessionTimeout    *string `json:"session_timeout"`
	IdleTimeout       *string `json:"idle_timeout"`
	KeepaliveTimeout  *string `json:"keepalive_timeout"`
	StatusAutorefresh *string `json:"status_autorefresh"`
	AddMacCookie      *string `json:"add_mac_cookie"`
	MacCookieTimeout  *string `json:"mac_cookie_timeout"`
	AddressList       *string `json:"address_list"`
	InsertQueueBefore *string `json:"insert_queue_before"`
	ParentQueue       *string `json:"parent_queue"`
	QueueType         *string `json:"queue_type"`
	TransparentProxy  *string `json:"transparent_proxy"`
	OnLogin           *string `json:"on_login"`
	OnLogout          *string `json:"on_logout"`
	Comment           *string `json:"comment"`
}

// Fields memuat semua properti opsional selain name.
func (input ProfileInput) Fields() []Field {
	return []Field{
		{"address-pool", input.AddressPool},
		{"shared-users", input.SharedUsers},
		{"rate-limit", input.RateLimit},
		{"session-timeout", input.SessionTimeout},
		{"idle-timeout", input.IdleTimeout},
		{"keepalive-timeout", input.KeepaliveTimeout},
		{"status-autorefresh", input.StatusAutorefresh},
		{"add-mac-cookie", input.AddMacCookie},
		{"mac-cookie-timeout", input.MacCookieTimeout},
		{"address-list", input.AddressList},
		{"insert-queue-before", input.InsertQueueBefore},
		{"parent-queue", input.ParentQueue},
		{"queue-type", input.QueueType},
		{"transparent-proxy", input.TransparentProxy},
		{"on-login", input.OnLogin},
		{"on-logout", input.OnLogout},
		{"comment", input.Comment},
	}
}

func ProfileFromRow(row map[string]string) Profile {
	return Profile{
		ID:                row[".id"],
		Name:              row["name"],
		AddressPool:       row["address-pool"],
		SharedUsers:       row["shared-users"],
		RateLimit:         row["rate-limit"],
		SessionTimeout:    row["session-timeout"],
		IdleTimeout:       row["idle-timeout"],
		KeepaliveTimeout:  row["keepalive-timeout"],
		StatusAutorefresh: row["status-autorefresh"],
		AddMacCookie:      row["add-mac-cookie"],
		MacCookieTimeout:  row["mac-cookie-timeout"],
		AddressList:       row["address-list"],
		InsertQueueBefore: row["insert-queue-before"],
		ParentQueue:       row["parent-queue"],
		QueueType:         row["queue-type"],
		TransparentProxy:  row["transparent-proxy"],
		OnLogin:           row["on-login"],
		OnLogout:          row["on-logout"],
		Comment:           row["comment"],
	}
}

func ListProfiles(userID uint, mikrotikID uint64) ([]Profile, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(profilePath+"/print", "=.proplist="+profileProperties)
	if err != nil {
		return nil, err
	}

	profiles := make([]Profile, 0, len(reply.Re))
	for _, row := range reply.Re {
		profiles = append(profiles, ProfileFromRow(row.Map))
	}

	// RouterOS mengembalikan data dari yang terlama; tampilkan yang terbaru lebih dulu.
	slices.Reverse(profiles)

	return profiles, nil
}

func ShowProfile(userID uint, mikrotikID uint64, id string) (*Profile, error) {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return nil, ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(profilePath+"/print", "?.id="+id, "=.proplist="+profileProperties)
	if err != nil {
		return nil, err
	}
	if len(reply.Re) == 0 {
		return nil, ErrProfileNotFound
	}

	profile := ProfileFromRow(reply.Re[0].Map)
	return &profile, nil
}

func CreateProfile(userID uint, mikrotikID uint64, input ProfileInput) error {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	args := append([]string{profilePath + "/add", "=name=" + input.Name}, CreateArgs(input.Fields())...)

	_, err = client.Run(args...)
	return MapDuplicate(err, ErrProfileExists)
}

func UpdateProfile(userID uint, mikrotikID uint64, id string, input ProfileInput) error {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return ErrInvalidInput
	}

	args := []string{profilePath + "/set", "=.id=" + id}

	if name := strings.TrimSpace(input.Name); name != "" {
		args = append(args, "=name="+name)
	}
	args = append(args, UpdateArgs(input.Fields())...)

	if len(args) == 2 {
		return ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	exists, err := itemExists(client, profilePath, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProfileNotFound
	}

	_, err = client.Run(args...)
	return MapDuplicate(err, ErrProfileExists)
}

func DeleteProfile(userID uint, mikrotikID uint64, id string) error {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	exists, err := itemExists(client, profilePath, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProfileNotFound
	}

	_, err = client.Run(profilePath+"/remove", "=.id="+id)
	return err
}
