package pppoe

import (
	"errors"
	"slices"
	"strings"
)

var ErrProfileNotFound = errors.New("PPP Profile not found")

type Profile struct {
	Name              string `json:"name"`
	LocalAddress      string `json:"local_address"`
	RemoteAddress     string `json:"remote_address"`
	OnlyOne           string `json:"only_one"`
	RateLimit         string `json:"rate_limit"`
	InsertQueueBefore string `json:"insert_queue_before"`
	ParentQueue       string `json:"parent_queue"`
	AddressList       string `json:"address_list"`
	OnUp              string `json:"on_up"`
	OnDown            string `json:"on_down"`
	Comment           string `json:"comment"`
}

// ProfileInput: for update, nil = unchanged and "" = clear; for create, empty values are skipped.
type ProfileInput struct {
	Name              string  `json:"name"`
	LocalAddress      *string `json:"local_address"`
	RemoteAddress     *string `json:"remote_address"`
	OnlyOne           *string `json:"only_one"`
	RateLimit         *string `json:"rate_limit"`
	InsertQueueBefore *string `json:"insert_queue_before"`
	ParentQueue       *string `json:"parent_queue"`
	AddressList       *string `json:"address_list"`
	OnUp              *string `json:"on_up"`
	OnDown            *string `json:"on_down"`
	Comment           *string `json:"comment"`
}

type profileArg struct {
	key   string
	value *string
}

// args lists every optional field with its RouterOS key, excluding name.
func (input ProfileInput) args() []profileArg {
	return []profileArg{
		{"local-address", input.LocalAddress},
		{"remote-address", input.RemoteAddress},
		{"only-one", input.OnlyOne},
		{"rate-limit", input.RateLimit},
		{"insert-queue-before", input.InsertQueueBefore},
		{"parent-queue", input.ParentQueue},
		{"address-list", input.AddressList},
		{"on-up", input.OnUp},
		{"on-down", input.OnDown},
		{"comment", input.Comment},
	}
}

const profileProperties = "name,local-address,remote-address,only-one,rate-limit,insert-queue-before,parent-queue,address-list,on-up,on-down,comment"

func profileFromRow(row map[string]string) Profile {
	return Profile{
		Name:              row["name"],
		LocalAddress:      row["local-address"],
		RemoteAddress:     row["remote-address"],
		OnlyOne:           row["only-one"],
		RateLimit:         row["rate-limit"],
		InsertQueueBefore: row["insert-queue-before"],
		ParentQueue:       row["parent-queue"],
		AddressList:       row["address-list"],
		OnUp:              row["on-up"],
		OnDown:            row["on-down"],
		Comment:           row["comment"],
	}
}

// GET: List semua profile
func ListProfiles(userID uint, mikrotikID uint64) ([]Profile, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(
		"/ppp/profile/print",
		"=.proplist="+profileProperties,
	)
	if err != nil {
		return nil, err
	}

	profiles := make([]Profile, 0, len(reply.Re))

	for _, row := range reply.Re {
		profiles = append(profiles, profileFromRow(row.Map))
	}

	OrderProfilesNewestFirst(profiles)

	return profiles, nil
}

// builtinProfiles adalah profil bawaan RouterOS; tidak dibuat user sehingga tidak dianggap "terbaru".
var builtinProfiles = []string{"default", "default-encryption"}

// OrderProfilesNewestFirst membalik urutan print RouterOS (terlama dulu) agar profil terbaru di atas,
// dan menaruh profil bawaan di paling bawah. Tanpa ini, default-encryption (yang muncul terakhir di print)
// justru menjadi baris paling atas setelah dibalik.
func OrderProfilesNewestFirst(profiles []Profile) {
	slices.Reverse(profiles)
	slices.SortStableFunc(profiles, func(a, b Profile) int {
		return builtinRank(a.Name) - builtinRank(b.Name)
	})
}

// builtinRank bernilai 0 untuk profil buatan user, dan 1 untuk profil bawaan.
func builtinRank(name string) int {
	if slices.Contains(builtinProfiles, name) {
		return 1
	}

	return 0
}

// GET: Detail profile berdasarkan nama
func ShowProfile(userID uint, mikrotikID uint64, name string) (*Profile, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(
		"/ppp/profile/print",
		"?name="+name,
		"=.proplist="+profileProperties,
	)
	if err != nil {
		return nil, err
	}

	if len(reply.Re) == 0 {
		return nil, ErrProfileNotFound
	}

	profile := profileFromRow(reply.Re[0].Map)

	return &profile, nil
}

// POST: Tambah profile
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

	args := []string{"/ppp/profile/add", "=name=" + input.Name}
	for _, arg := range input.args() {
		if v := optionalValue(arg.value); v != "" {
			args = append(args, "="+arg.key+"="+v)
		}
	}

	_, err = client.Run(args...)

	return MapDeviceError(err)
}

// PUT: Update profile
func UpdateProfile(
	userID uint,
	mikrotikID uint64,
	name string,
	input ProfileInput,
) error {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	reply, err := client.Run(
		"/ppp/profile/print",
		"?name="+name,
		"=.proplist=.id",
	)
	if err != nil {
		return err
	}

	if len(reply.Re) == 0 {
		return ErrProfileNotFound
	}

	profileID := reply.Re[0].Map[".id"]

	args := []string{
		"/ppp/profile/set",
		"=.id=" + profileID,
	}

	if name := strings.TrimSpace(input.Name); name != "" {
		args = append(args, "=name="+name)
	}
	for _, arg := range input.args() {
		if arg.value != nil {
			args = append(args, "="+arg.key+"="+strings.TrimSpace(*arg.value))
		}
	}

	if len(args) == 2 {
		return ErrInvalidInput
	}

	_, err = client.Run(args...)

	return MapDeviceError(err)
}

// DELETE: Hapus profile
func DeleteProfile(userID uint, mikrotikID uint64, name string) error {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	reply, err := client.Run(
		"/ppp/profile/print",
		"?name="+name,
		"=.proplist=.id",
	)
	if err != nil {
		return err
	}

	if len(reply.Re) == 0 {
		return ErrProfileNotFound
	}

	_, err = client.Run(
		"/ppp/profile/remove",
		"=.id="+reply.Re[0].Map[".id"],
	)

	return err
}
