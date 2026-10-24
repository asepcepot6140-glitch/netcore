package pppoe

import (
	"errors"
	"net/http"

	"github.com/asepcepot6140-glitch/netcore/internal/services/pppoe"

	"github.com/gin-gonic/gin"
)

func ListActiveConnections(c *gin.Context) {
	mikrotikID, err := pppoe.ParseMikrotikID(c.Query("mikrotik_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid mikrotik_id",
		})
		return
	}

	connections, err := pppoe.ListActiveConnections(
		c.GetUint("user_id"),
		mikrotikID,
	)
	if err != nil {
		if errors.Is(err, pppoe.ErrMikrotikNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "MikroTik not found",
			})
			return
		}

		upstreamError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": connections,
	})
}

// DELETE /api/pppoe/active/:id?mikrotik_id=1
func RemoveActiveConnection(c *gin.Context) {
	mikrotikID, err := pppoe.ParseMikrotikID(c.Query("mikrotik_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid mikrotik_id",
		})
		return
	}

	err = pppoe.RemoveActiveConnection(
		c.GetUint("user_id"),
		mikrotikID,
		c.Param("id"),
	)
	if err != nil {
		switch {
		case errors.Is(err, pppoe.ErrMikrotikNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "MikroTik not found",
			})
		case errors.Is(err, pppoe.ErrActiveConnectionNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Active PPP connection not found",
			})
		case errors.Is(err, pppoe.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid connection ID",
			})
		default:
			upstreamError(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Active PPP connection disconnected successfully",
	})
}
