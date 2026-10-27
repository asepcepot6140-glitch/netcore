package genieacs

import (
	"net/http"
	"strconv"

	"github.com/asepcepot6140-glitch/netcore/internal/services/genieacs"
	"github.com/gin-gonic/gin"
)

func GetLAN(c *gin.Context) {
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

	lan, err := genieacs.GetLAN(c.Request.Context(), item, deviceID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": "Failed to fetch LAN from GenieACS"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": lan})
}
