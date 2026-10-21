package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/services/whatsapp"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidOTP    = errors.New("OTP tidak valid atau sudah kedaluwarsa")
	ErrOTPTooMany    = errors.New("batas percobaan OTP tercapai")
	ErrOTPWait       = errors.New("tunggu sebelum meminta OTP lagi")
	ErrInvalidReset  = errors.New("permintaan reset password tidak valid")
	ErrPasswordShort = errors.New("password minimal 8 karakter")
)

const (
	otpExpiry      = 5 * time.Minute
	otpResendWait  = 1 * time.Minute
	maxOTPAttempts = 5
)

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashOTP(otp string) string {
	sum := sha256.Sum256([]byte(otp))
	return fmt.Sprintf("%x", sum[:])
}

// RequestPasswordReset mengirim OTP ke nomor WhatsApp yang terdaftar.
func RequestPasswordReset(phone string) error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ErrInvalidReset
	}

	var user models.User
	err := config.DB.Where("phone = ?", phone).First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Jangan mengungkapkan apakah nomor terdaftar.
		return nil
	}
	if err != nil {
		return err
	}

	now := time.Now()

	var latest models.PasswordResetOTP
	err = config.DB.
		Where("user_id = ?", user.ID).
		Order("created_at DESC").
		First(&latest).Error

	if err == nil && now.Sub(latest.CreatedAt) < otpResendWait {
		return ErrOTPWait
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	otp, err := generateOTP()
	if err != nil {
		return err
	}

	otpRecord := models.PasswordResetOTP{
		UserID:    user.ID,
		OTPHash:   hashOTP(otp),
		ExpiresAt: now.Add(otpExpiry),
		Attempts:  0,
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		// OTP lama yang belum digunakan tidak boleh dipakai lagi.
		if err := tx.Model(&models.PasswordResetOTP{}).
			Where("user_id = ? AND used_at IS NULL", user.ID).
			Update("used_at", now).Error; err != nil {
			return err
		}

		return tx.Create(&otpRecord).Error
	})
	if err != nil {
		return err
	}

	message := fmt.Sprintf(
		"Kode OTP reset password kamu: %s\nBerlaku selama 5 menit.\nJangan berikan kode ini kepada siapa pun.",
		otp,
	)

	if err := whatsapp.SendMessage(phone, message); err != nil {
		// OTP yang gagal dikirim tidak dapat digunakan.
		_ = config.DB.Model(&otpRecord).Update("used_at", time.Now()).Error
		return err
	}

	return nil
}

// VerifyPasswordResetOTP memeriksa OTP tanpa menggunakannya.
// OTP baru ditandai terpakai saat password berhasil diganti.
func VerifyPasswordResetOTP(phone, otp string) error {
	phone = strings.TrimSpace(phone)
	otp = strings.TrimSpace(otp)

	if phone == "" || len(otp) != 6 {
		return ErrInvalidOTP
	}

	var user models.User
	if err := config.DB.Where("phone = ?", phone).First(&user).Error; err != nil {
		return ErrInvalidOTP
	}

	var record models.PasswordResetOTP
	err := config.DB.
		Where(
			"user_id = ? AND used_at IS NULL AND expires_at > ?",
			user.ID,
			time.Now(),
		).
		Order("created_at DESC").
		First(&record).Error

	if err != nil {
		return ErrInvalidOTP
	}

	if record.Attempts >= maxOTPAttempts {
		return ErrOTPTooMany
	}

	if record.OTPHash != hashOTP(otp) {
		if err := config.DB.Model(&record).
			Update("attempts", record.Attempts+1).Error; err != nil {
			return err
		}

		if record.Attempts+1 >= maxOTPAttempts {
			return ErrOTPTooMany
		}

		return ErrInvalidOTP
	}

	return nil
}

// ResetPassword memverifikasi OTP, mengganti password,
// menggunakan OTP, dan mencabut seluruh refresh token user.
func ResetPassword(phone, otp, newPassword string) error {
	phone = strings.TrimSpace(phone)
	otp = strings.TrimSpace(otp)

	if phone == "" || len(otp) != 6 {
		return ErrInvalidOTP
	}

	if len(newPassword) < 8 || len(newPassword) > 72 {
		return ErrPasswordShort
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(newPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	return config.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.
			Where("phone = ?", phone).
			First(&user).Error; err != nil {
			return ErrInvalidOTP
		}

		var record models.PasswordResetOTP
		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(
				"user_id = ? AND used_at IS NULL AND expires_at > ?",
				user.ID,
				time.Now(),
			).
			Order("created_at DESC").
			First(&record).Error

		if err != nil {
			return ErrInvalidOTP
		}

		if record.Attempts >= maxOTPAttempts {
			return ErrOTPTooMany
		}

		if record.OTPHash != hashOTP(otp) {
			if err := tx.Model(&record).
				Update("attempts", record.Attempts+1).Error; err != nil {
				return err
			}

			if record.Attempts+1 >= maxOTPAttempts {
				return ErrOTPTooMany
			}

			return ErrInvalidOTP
		}

		now := time.Now()

		if err := tx.Model(&user).
			Update("password", string(passwordHash)).Error; err != nil {
			return err
		}

		if err := tx.Model(&record).
			Update("used_at", now).Error; err != nil {
			return err
		}

		// Cabut seluruh refresh token lama.
		if err := tx.
			Where("user_id = ?", user.ID).
			Delete(&models.RefreshToken{}).Error; err != nil {
			return err
		}

		// OTP lain yang masih aktif juga tidak boleh digunakan.
		return tx.Model(&models.PasswordResetOTP{}).
			Where("user_id = ? AND id <> ? AND used_at IS NULL", user.ID, record.ID).
			Update("used_at", now).Error
	})
}
