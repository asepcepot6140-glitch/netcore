package pppoe

import (
	"errors"
	"net/http"

	"github.com/asepcepot6140-glitch/netcore/internal/services/pppoe"

	"github.com/gin-gonic/gin"
)

func ListOfflineConnections(c *gin.Context) {
	mikrotikID, err := pppoe.ParseMikrotikID(c.Query("mikrotik_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid mikrotik_id",
		})
		return
	}

	connections, err := pppoe.ListOfflineConnections(
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
