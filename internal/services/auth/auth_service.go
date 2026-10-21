package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmailExists  = errors.New("email sudah terdaftar")
	ErrPhoneExists  = errors.New("nomor WhatsApp sudah terdaftar")
	ErrInvalidLogin = errors.New("email atau password salah")
	ErrInvalidToken = errors.New("refresh token tidak valid")
	ErrInvalidInput = errors.New("input tidak valid")
	ErrUserNotFound = errors.New("user tidak ditemukan")
)

type RegisterInput struct {
	Name     string
	Email    string
	Phone    string
	Password string
}

type LoginResult struct {
	User         models.User
	AccessToken  string
	RefreshToken string
}

type RefreshResult struct {
	AccessToken  string
	RefreshToken string
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func Register(input RegisterInput) (*models.User, error) {
	name := strings.TrimSpace(input.Name)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	phone := strings.TrimSpace(input.Phone)

	if len(name) < 2 || len(name) > 100 ||
		len(phone) < 10 || len(phone) > 20 ||
		len(input.Password) < 8 {
		return nil, ErrInvalidInput
	}

	var existing models.User

	err := config.DB.
		Where("email = ? OR phone = ?", email, phone).
		First(&existing).Error

	if err == nil {
		if existing.Email == email {
			return nil, ErrEmailExists
		}
		return nil, ErrPhoneExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Name:     name,
		Email:    email,
		Phone:    phone,
		Password: string(passwordHash),
	}

	if err := config.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func Login(identifier, password string) (*LoginResult, error) {
	identifier = strings.TrimSpace(identifier)

	var user models.User

	err := config.DB.
		Where("email = ? OR phone = ?", strings.ToLower(identifier), identifier).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidLogin
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(password),
	); err != nil {
		return nil, ErrInvalidLogin
	}

	accessToken, err := utils.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	token := models.RefreshToken{
		UserID:    user.ID,
		TokenHash: hashToken(refreshToken),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := config.DB.Create(&token).Error; err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func Refresh(token string) (*RefreshResult, error) {
	claims, err := utils.ParseToken(token)
	if err != nil || claims.Type != "refresh" || claims.UserID == 0 {
		return nil, ErrInvalidToken
	}

	tokenHash := hashToken(token)
	now := time.Now()

	var stored models.RefreshToken

	err = config.DB.
		Where("token_hash = ?", tokenHash).
		First(&stored).Error

	if err != nil || stored.UserID != claims.UserID ||
		!stored.ExpiresAt.After(now) {
		return nil, ErrInvalidToken
	}

	accessToken, err := utils.GenerateAccessToken(claims.UserID)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := utils.GenerateRefreshToken(claims.UserID)
	if err != nil {
		return nil, err
	}

	errInvalidToken := ErrInvalidToken

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where(
				"token_hash = ? AND user_id = ? AND expires_at > ?",
				tokenHash,
				claims.UserID,
				now,
			).
			Delete(&models.RefreshToken{})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errInvalidToken
		}

		return tx.Create(&models.RefreshToken{
			UserID:    claims.UserID,
			TokenHash: hashToken(newRefreshToken),
			ExpiresAt: now.Add(7 * 24 * time.Hour),
		}).Error
	})

	if err != nil {
		return nil, err
	}

	return &RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func Logout(userID uint) error {
	return config.DB.
		Where("user_id = ?", userID).
		Delete(&models.RefreshToken{}).Error
}

func Me(userID uint) (*models.User, error) {
	var user models.User

	if err := config.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}
