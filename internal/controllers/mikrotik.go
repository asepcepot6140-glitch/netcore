package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MikrotikRequest struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
	Port     int    `json:"port"`
}

func ListMikrotik(c *gin.Context) {
	userID := c.GetUint("user_id")

	mikrotiks, err := services.ListMikrotik(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to fetch Mikrotik data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": mikrotiks})
}

func CreateMikrotik(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req MikrotikRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid Mikrotik data",
		})
		return
	}

	mikrotik, err := services.CreateMikrotik(userID, services.MikrotikInput{
		Name:     req.Name,
		Host:     req.Host,
		Username: req.Username,
		Password: req.Password,
		Port:     req.Port,
	})

	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidPort),
			errors.Is(err, services.ErrInvalidPassword):
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})

		case strings.Contains(err.Error(), "is required"),
			strings.Contains(err.Error(), "are required"):
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Failed to save Mikrotik",
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Mikrotik added successfully",
		"data":    mikrotik,
	})
}

func ShowMikrotik(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid Mikrotik ID",
		})
		return
	}

	mikrotik, err := services.ShowMikrotik(userID, id)
	if err != nil {
		if errors.Is(err, services.ErrMikrotikNotFound) ||
			errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Mikrotik not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to fetch Mikrotik data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": mikrotik})
}

func UpdateMikrotik(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid Mikrotik ID",
		})
		return
	}

	var req MikrotikRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid Mikrotik data",
		})
		return
	}

	mikrotik, err := services.UpdateMikrotik(userID, id, services.MikrotikInput{
		Name:     req.Name,
		Host:     req.Host,
		Username: req.Username,
		Password: req.Password,
		Port:     req.Port,
	})

	if err != nil {
		switch {
		case errors.Is(err, services.ErrMikrotikNotFound),
			errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Mikrotik not found",
			})

		case errors.Is(err, services.ErrInvalidPort),
			errors.Is(err, services.ErrNoChanges):
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Failed to update Mikrotik",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Mikrotik updated successfully",
		"data":    mikrotik,
	})
}

func DeleteMikrotik(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid Mikrotik ID",
		})
		return
	}

	err = services.DeleteMikrotik(userID, id)
	if err != nil {
		if errors.Is(err, services.ErrMikrotikNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Mikrotik not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to delete Mikrotik",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Mikrotik deleted successfully",
	})
}
