package services

import (
	"errors"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"gorm.io/gorm"
)

var (
	ErrMikrotikNotFound = errors.New("Mikrotik not found")
	ErrInvalidPort      = errors.New("Invalid port")
	ErrInvalidPassword  = errors.New("Password is required")
	ErrNoChanges        = errors.New("No changes were made")
)

type MikrotikInput struct {
	Name     string
	Host     string
	Username string
	Password string
	Port     int
}

func ListMikrotik(userID uint) ([]models.Mikrotik, error) {
	var mikrotiks []models.Mikrotik

	err := config.DB.
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&mikrotiks).Error

	return mikrotiks, err
}

func CreateMikrotik(userID uint, input MikrotikInput) (*models.Mikrotik, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Host = strings.TrimSpace(input.Host)
	input.Username = strings.TrimSpace(input.Username)

	if input.Name == "" || input.Host == "" || input.Username == "" {
		return nil, errors.New("Name, host, and username are required")
	}

	if input.Password == "" {
		return nil, ErrInvalidPassword
	}

	if input.Port == 0 {
		input.Port = 8728
	}

	if input.Port < 1 || input.Port > 65535 {
		return nil, ErrInvalidPort
	}

	encryptedPassword, err := utils.EncryptMikrotikPassword(input.Password)
	if err != nil {
		return nil, err
	}

	mikrotik := models.Mikrotik{
		UserID:            userID,
		Name:              input.Name,
		Host:              input.Host,
		Username:          input.Username,
		PasswordEncrypted: encryptedPassword,
		Port:              input.Port,
	}

	if err := config.DB.Create(&mikrotik).Error; err != nil {
		return nil, err
	}

	return &mikrotik, nil
}

func ShowMikrotik(userID uint, id uint64) (*models.Mikrotik, error) {
	var mikrotik models.Mikrotik

	err := config.DB.
		Where("id = ? AND user_id = ?", id, userID).
		First(&mikrotik).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMikrotikNotFound
	}

	if err != nil {
		return nil, err
	}

	return &mikrotik, nil
}

func UpdateMikrotik(userID uint, id uint64, input MikrotikInput) (*models.Mikrotik, error) {
	mikrotik, err := ShowMikrotik(userID, id)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}

	name := strings.TrimSpace(input.Name)
	host := strings.TrimSpace(input.Host)
	username := strings.TrimSpace(input.Username)

	if name != "" {
		updates["name"] = name
	}

	if host != "" {
		updates["host"] = host
	}

	if username != "" {
		updates["username"] = username
	}

	if input.Port != 0 {
		if input.Port < 1 || input.Port > 65535 {
			return nil, ErrInvalidPort
		}

		updates["port"] = input.Port
	}

	if input.Password != "" {
		encryptedPassword, err := utils.EncryptMikrotikPassword(input.Password)
		if err != nil {
			return nil, err
		}

		updates["password_encrypted"] = encryptedPassword
	}

	if len(updates) == 0 {
		return nil, ErrNoChanges
	}

	if err := config.DB.Model(mikrotik).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := config.DB.First(mikrotik, mikrotik.ID).Error; err != nil {
		return nil, err
	}

	return mikrotik, nil
}

func DeleteMikrotik(userID uint, id uint64) error {
	result := config.DB.
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Mikrotik{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrMikrotikNotFound
	}

	return nil
}
