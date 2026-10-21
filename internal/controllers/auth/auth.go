package auth

import (
	"errors"
	"net/http"

	"github.com/asepcepot6140-glitch/netcore/internal/services/auth"

	"github.com/gin-gonic/gin"
)

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone" binding:"required,min=10,max=20"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Password   string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Data registrasi tidak valid",
			"error":   err.Error(),
		})
		return
	}

	user, err := auth.Register(auth.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Data registrasi tidak valid",
			})
		case errors.Is(err, auth.ErrEmailExists):
			c.JSON(http.StatusConflict, gin.H{
				"message": "Email sudah terdaftar",
			})
		case errors.Is(err, auth.ErrPhoneExists):
			c.JSON(http.StatusConflict, gin.H{
				"message": "Nomor WhatsApp sudah terdaftar",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Gagal membuat user",
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registrasi berhasil",
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"phone": user.Phone,
		},
	})
}

func Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Data login tidak valid",
		})
		return
	}

	identifier := req.Identifier
	if identifier == "" {
		identifier = req.Email
	}
	if identifier == "" {
		identifier = req.Phone
	}

	if identifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Email atau nomor WhatsApp diperlukan",
		})
		return
	}

	result, err := auth.Login(identifier, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidLogin) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Email atau password salah",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal memproses login",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login berhasil",
		"data": gin.H{
			"user": gin.H{
				"id":    result.User.ID,
				"name":  result.User.Name,
				"email": result.User.Email,
				"phone": result.User.Phone,
			},
			"access_token":  result.AccessToken,
			"refresh_token": result.RefreshToken,
			"token_type":    "Bearer",
			"expires_in":    900,
		},
	})
}

func Refresh(c *gin.Context) {
	var req RefreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Refresh token diperlukan",
		})
		return
	}

	result, err := auth.Refresh(req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Refresh token sudah tidak valid",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal memperbarui refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

func Logout(c *gin.Context) {
	userID := c.GetUint("user_id")

	if err := auth.Logout(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal logout",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logout berhasil",
	})
}

func Me(c *gin.Context) {
	userID := c.GetUint("user_id")

	user, err := auth.Me(userID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "User tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal mengambil data user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"phone": user.Phone,
	})
}
