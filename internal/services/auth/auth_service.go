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
	ErrEmailExists  = errors.New("email is already registered")
	ErrPhoneExists  = errors.New("WhatsApp number is already registered")
	ErrInvalidLogin = errors.New("invalid email or password")
	ErrInvalidToken = errors.New("invalid refresh token")
	ErrInvalidInput = errors.New("invalid input")
	ErrUserNotFound = errors.New("user not found")
)

// maxEmailLength sesuai ukuran kolom users.email.
const maxEmailLength = 150

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

// dummyHash hanya untuk menyamakan waktu login user yang tidak ada; tidak cocok dengan password nyata.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func Register(input RegisterInput) (*models.User, error) {
	name := strings.TrimSpace(input.Name)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	phone := utils.NormalizePhone(input.Phone)

	if len(name) < 2 || len(name) > 100 ||
		len(email) == 0 || len(email) > maxEmailLength ||
		phone == "" ||
		len(input.Password) < 8 || len(input.Password) > 72 {
		return nil, ErrInvalidInput
	}

	if err := findConflict(email, phone); err != nil {
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
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			// Request paralel lolos pengecekan awal; tentukan kolom yang bentrok.
			if conflict := findConflict(email, phone); conflict != nil {
				return nil, conflict
			}
			return nil, ErrEmailExists
		}
		return nil, err
	}

	return &user, nil
}

// findConflict mengembalikan ErrEmailExists/ErrPhoneExists jika email atau nomor sudah terdaftar.
func findConflict(email, phone string) error {
	var existing models.User

	err := config.DB.
		Where("email = ? OR phone = ?", email, phone).
		First(&existing).Error

	if err == nil {
		if existing.Email == email {
			return ErrEmailExists
		}
		return ErrPhoneExists
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return err
}

func Login(identifier, password string) (*LoginResult, error) {
	identifier = strings.TrimSpace(identifier)

	// Identifier berisi "@" dianggap email; selain itu dicoba sebagai nomor WhatsApp.
	phone := identifier
	if !strings.Contains(identifier, "@") {
		if normalized := utils.NormalizePhone(identifier); normalized != "" {
			phone = normalized
		}
	}

	var user models.User

	err := config.DB.
		Where("email = ? OR phone = ?", strings.ToLower(identifier), phone).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Jalankan bcrypt agar waktu respons tidak membedakan user yang tidak ada.
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
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
