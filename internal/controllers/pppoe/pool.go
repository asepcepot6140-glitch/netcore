package pppoe

import (
	"errors"
	"net/http"

	"github.com/asepcepot6140-glitch/netcore/internal/services/pppoe"

	"github.com/gin-gonic/gin"
)

func ListPools(c *gin.Context) {
	mikrotikID, err := pppoe.ParseMikrotikID(c.Query("mikrotik_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "mikrotik_id tidak valid",
		})
		return
	}

	pools, err := pppoe.ListPools(c.GetUint("user_id"), mikrotikID)
	if err != nil {
		if errors.Is(err, pppoe.ErrMikrotikNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "MikroTik tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": pools,
	})
}
