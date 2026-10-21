package auth

import (
	"errors"
	"net/http"

	"github.com/asepcepot6140-glitch/netcore/internal/services/auth"

	"github.com/gin-gonic/gin"
)

type ForgotPasswordRequest struct {
	Phone string `json:"phone" binding:"required,min=10,max=20"`
}

type VerifyOTPRequest struct {
	Phone string `json:"phone" binding:"required,min=10,max=20"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

type ResetPasswordRequest struct {
	Phone       string `json:"phone" binding:"required,min=10,max=20"`
	OTP         string `json:"otp" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

func ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Nomor WhatsApp tidak valid",
		})
		return
	}

	err := auth.RequestPasswordReset(req.Phone)
	if err != nil {
		if errors.Is(err, auth.ErrOTPWait) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"message": "Tunggu satu menit sebelum meminta OTP lagi",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal memproses permintaan OTP",
		})
		return
	}

	// Respons sama meskipun nomor tidak terdaftar.
	c.JSON(http.StatusOK, gin.H{
		"message": "Jika nomor terdaftar, kode OTP akan dikirim melalui WhatsApp",
	})
}

func VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Nomor WhatsApp atau OTP tidak valid",
		})
		return
	}

	err := auth.VerifyPasswordResetOTP(req.Phone, req.OTP)
	if err != nil {
		status := http.StatusBadRequest
		message := "OTP tidak valid atau sudah kedaluwarsa"

		if errors.Is(err, auth.ErrOTPTooMany) {
			status = http.StatusTooManyRequests
			message = "Batas percobaan OTP tercapai"
		}

		c.JSON(status, gin.H{"message": message})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OTP valid",
	})
}

func ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Data reset password tidak valid",
		})
		return
	}

	err := auth.ResetPassword(
		req.Phone,
		req.OTP,
		req.NewPassword,
	)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidOTP):
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "OTP tidak valid atau sudah kedaluwarsa",
			})
		case errors.Is(err, auth.ErrOTPTooMany):
			c.JSON(http.StatusTooManyRequests, gin.H{
				"message": "Batas percobaan OTP tercapai",
			})
		case errors.Is(err, auth.ErrPasswordShort):
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Password harus 8 sampai 72 byte",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Gagal mengganti password",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password berhasil diganti. Silakan login kembali.",
	})
}
