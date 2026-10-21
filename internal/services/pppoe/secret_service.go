package pppoe

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"github.com/go-routeros/routeros/v3"
	"gorm.io/gorm"
)

var (
	ErrMikrotikNotFound = errors.New("MikroTik tidak ditemukan")
	ErrSecretNotFound   = errors.New("PPP Secret tidak ditemukan")
	ErrInvalidInput     = errors.New("input tidak valid")
)

type SecretInput struct {
	Name          string `json:"name"`
	Password      string `json:"password"`
	Service       string `json:"service"`
	Profile       string `json:"profile"`
	LocalAddress  string `json:"local_address"`
	RemoteAddress string `json:"remote_address"`
	Comment       string `json:"comment"`
}

type Secret struct {
	ID                   string `json:".id"`
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

func connect(userID uint, mikrotikID uint64) (*routeros.Client, error) {
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
		return nil, fmt.Errorf("gagal mendekripsi password MikroTik: %w", err)
	}

	address := fmt.Sprintf("%s:%d", mikrotik.Host, mikrotik.Port)

	client, err := routeros.Dial(
		address,
		mikrotik.Username,
		password,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal terhubung ke MikroTik: %w", err)
	}

	return client, nil
}

func ListSecrets(userID uint, mikrotikID uint64) ([]Secret, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(
		"/ppp/secret/print",
		"=.proplist=.id,name,disabled,service,caller-id,password,profile,remote-address,local-address,routes,ipv6-routes,limit-bytes-in,limit-bytes-out,last-logged-out,last-caller-id,last-disconnect-reason,comment",
	)
	if err != nil {
		return nil, err
	}
	secrets := make([]Secret, 0, len(reply.Re))

	for _, row := range reply.Re {
		secrets = append(secrets, Secret{
			ID:                   row.Map[".id"],
			Disabled:             row.Map["disabled"] == "true",
			Name:                 row.Map["name"],
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

	return secrets, nil
}

func ShowSecret(userID uint, mikrotikID uint64, secretID string) (*Secret, error) {
	secrets, err := ListSecrets(userID, mikrotikID)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		if secret.ID == secretID {
			return &secret, nil
		}
	}

	return nil, ErrSecretNotFound
}

func CreateSecret(userID uint, mikrotikID uint64, input SecretInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Service = strings.TrimSpace(input.Service)
	input.Profile = strings.TrimSpace(input.Profile)

	if input.Name == "" || input.Password == "" {
		return ErrInvalidInput
	}

	if input.Service == "" {
		input.Service = "pppoe"
	}
	if input.Profile == "" {
		input.Profile = "default"
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.Run(
		"/ppp/secret/add",
		"=name="+input.Name,
		"=password="+input.Password,
		"=service="+input.Service,
		"=profile="+input.Profile,
		"=local-address="+input.LocalAddress,
		"=remote-address="+input.RemoteAddress,
		"=comment="+input.Comment,
	)
	return err
}

func UpdateSecret(userID uint, mikrotikID uint64, secretID string, input SecretInput) error {
	secretID = strings.TrimSpace(secretID)
	if secretID == "" || !strings.HasPrefix(secretID, "*") {
		return ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	reply, err := client.Run(
		"/ppp/secret/print",
		"?.id="+secretID,
	)
	if err != nil {
		return err
	}
	if len(reply.Re) == 0 {
		return ErrSecretNotFound
	}

	args := []string{"/ppp/secret/set", "=.id=" + secretID}

	if name := strings.TrimSpace(input.Name); name != "" {
		args = append(args, "=name="+name)
	}
	if input.Password != "" {
		args = append(args, "=password="+input.Password)
	}
	if service := strings.TrimSpace(input.Service); service != "" {
		args = append(args, "=service="+service)
	}
	if profile := strings.TrimSpace(input.Profile); profile != "" {
		args = append(args, "=profile="+profile)
	}
	if input.LocalAddress != "" {
		args = append(args, "=local-address="+input.LocalAddress)
	}
	if input.RemoteAddress != "" {
		args = append(args, "=remote-address="+input.RemoteAddress)
	}
	if input.Comment != "" {
		args = append(args, "=comment="+input.Comment)
	}

	if len(args) == 2 {
		return ErrInvalidInput
	}

	_, err = client.Run(args...)
	return err
}

func DeleteSecret(userID uint, mikrotikID uint64, secretID string) error {
	secretID = strings.TrimSpace(secretID)
	if secretID == "" || !strings.HasPrefix(secretID, "*") {
		return ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	reply, err := client.Run(
		"/ppp/secret/print",
		"?.id="+secretID,
	)
	if err != nil {
		return err
	}
	if len(reply.Re) == 0 {
		return ErrSecretNotFound
	}

	_, err = client.Run(
		"/ppp/secret/remove",
		"=.id="+secretID,
	)
	return err
}

func ParseMikrotikID(value string) (uint64, error) {
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, ErrInvalidInput
	}
	return id, nil
}
