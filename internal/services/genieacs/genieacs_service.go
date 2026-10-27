package genieacs

import (
	"errors"
	"fmt"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
	"github.com/asepcepot6140-glitch/netcore/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound     = errors.New("GenieACS configuration not found")
	ErrInvalidInput = errors.New("invalid GenieACS data")
	ErrLimitReached = fmt.Errorf("GenieACS server limit reached (maximum %d per account)", MaxGenieACSPerUser)
)

// MaxGenieACSPerUser membatasi jumlah server GenieACS per akun.
const MaxGenieACSPerUser = 20

type GenieACSInput struct {
	Name string
	Host string
	Port int
}

func validateInput(input GenieACSInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Host = strings.TrimSpace(input.Host)

	if len(input.Name) < 2 || len(input.Name) > 100 {
		return ErrInvalidInput
	}

	if input.Host == "" || len(input.Host) > 255 {
		return ErrInvalidInput
	}

	if input.Port < 1 || input.Port > 65535 {
		return ErrInvalidInput
	}

	return nil
}

func List(userID uint) ([]models.GenieACS, error) {
	var items []models.GenieACS

	err := config.DB.
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&items).Error

	return items, err
}

func Show(userID, id uint) (*models.GenieACS, error) {
	var item models.GenieACS

	err := config.DB.
		Where("id = ? AND user_id = ?", id, userID).
		First(&item).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	return &item, nil
}

func Create(userID uint, input GenieACSInput) (*models.GenieACS, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Host = strings.TrimSpace(input.Host)

	if err := validateInput(input); err != nil {
		return nil, err
	}

	item := models.GenieACS{
		UserID: userID,
		Name:   input.Name,
		Host:   input.Host,
		Port:   input.Port,
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// Kunci baris user agar dua request paralel tidak sama-sama lolos batas.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").First(&models.User{}, userID).Error; err != nil {
			return err
		}

		var count int64
		if err := tx.Model(&models.GenieACS{}).
			Where("user_id = ?", userID).
			Count(&count).Error; err != nil {
			return err
		}

		if count >= MaxGenieACSPerUser {
			return ErrLimitReached
		}

		return tx.Create(&item).Error
	})
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func Update(
	userID, id uint,
	input GenieACSInput,
) (*models.GenieACS, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Host = strings.TrimSpace(input.Host)

	if err := validateInput(input); err != nil {
		return nil, err
	}

	var item models.GenieACS

	err := config.DB.
		Where("id = ? AND user_id = ?", id, userID).
		First(&item).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	item.Name = input.Name
	item.Host = input.Host
	item.Port = input.Port

	if err := config.DB.Save(&item).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func Delete(userID, id uint) error {
	result := config.DB.
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.GenieACS{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
