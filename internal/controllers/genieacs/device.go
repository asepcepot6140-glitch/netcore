package genieacs

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/asepcepot6140-glitch/netcore/internal/services/genieacs"
	"github.com/gin-gonic/gin"
)

func ListDevices(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid GenieACS ID"})
		return
	}

	item, err := genieacs.Show(c.GetUint("user_id"), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "GenieACS configuration not found"})
		return
	}

	devices, err := genieacs.GetDevices(c.Request.Context(), item)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": "Failed to fetch devices from GenieACS"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(devices),
		"data":  devices,
	})
}

func SummonDevice(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid GenieACS ID"})
		return
	}

	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Device ID is required"})
		return
	}

	item, err := genieacs.Show(c.GetUint("user_id"), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "GenieACS configuration not found"})
		return
	}

	scope := c.Query("scope")
	var applied bool
	if scope == "" {
		applied, err = genieacs.SummonDevice(c.Request.Context(), item, deviceID)
	} else {
		switch scope {
		case "wan", "lan", "wlan":
			applied, err = genieacs.SummonDeviceObject(
				c.Request.Context(),
				item,
				deviceID,
				scope,
			)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid summon scope"})
			return
		}
	}
	if err != nil {
		log.Printf("genieacs summon error: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"message": "Failed to send refresh request to GenieACS"})
		return
	}

	message := "Object refresh request sent successfully"
	if !applied {
		message = "Request timeout"
	}
	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"applied": applied,
	})
}

func DeleteDevice(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid GenieACS ID"})
		return
	}

	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Device ID is required"})
		return
	}

	item, err := genieacs.Show(c.GetUint("user_id"), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "GenieACS configuration not found"})
		return
	}

	if err := genieacs.DeleteDevice(c.Request.Context(), item, deviceID); err != nil {
		if errors.Is(err, genieacs.ErrDeviceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Device not found"})
			return
		}
		log.Printf("genieacs delete device error: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"message": "Failed to delete device from GenieACS"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device deleted successfully"})
}
