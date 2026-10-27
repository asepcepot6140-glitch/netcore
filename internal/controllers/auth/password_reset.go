package auth

import (
	"errors"
	"net/http"

	"github.com/asepcepot6140-glitch/netcore/internal/services/auth"

	"github.com/gin-gonic/gin"
)

type ForgotPasswordRequest struct {
	Phone string `json:"phone" binding:"required,max=30"`
}

type VerifyOTPRequest struct {
	Phone string `json:"phone" binding:"required,max=30"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

type ResetPasswordRequest struct {
	Phone       string `json:"phone" binding:"required,max=30"`
	OTP         string `json:"otp" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

func ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid WhatsApp number",
		})
		return
	}

	err := auth.RequestPasswordReset(req.Phone)
	if err != nil {
		if errors.Is(err, auth.ErrOTPWait) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"message": "Please wait one minute before requesting another OTP",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to process OTP request",
		})
		return
	}

	// Respons sama meskipun nomor tidak terdaftar.
	c.JSON(http.StatusOK, gin.H{
		"message": "If the number is registered, an OTP code will be sent via WhatsApp",
	})
}

func VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid WhatsApp number or OTP",
		})
		return
	}

	err := auth.VerifyPasswordResetOTP(req.Phone, req.OTP)
	if err != nil {
		status := http.StatusBadRequest
		message := "Invalid or expired OTP"

		if errors.Is(err, auth.ErrOTPTooMany) {
			status = http.StatusTooManyRequests
			message = "OTP attempt limit reached"
		}

		c.JSON(status, gin.H{"message": message})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OTP verified",
	})
}

func ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid password reset data",
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
				"message": "Invalid or expired OTP",
			})
		case errors.Is(err, auth.ErrOTPTooMany):
			c.JSON(http.StatusTooManyRequests, gin.H{
				"message": "OTP attempt limit reached",
			})
		case errors.Is(err, auth.ErrPasswordShort):
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Password must be between 8 and 72 bytes",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Failed to reset password",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password has been reset. Please sign in again.",
	})
}
