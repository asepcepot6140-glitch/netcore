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
			"error": "mikrotik_id tidak valid",
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
		"data": connections,
	})
}

// DELETE /api/pppoe/active/:id?mikrotik_id=1
func RemoveActiveConnection(c *gin.Context) {
	mikrotikID, err := pppoe.ParseMikrotikID(c.Query("mikrotik_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "mikrotik_id tidak valid",
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
				"error": "MikroTik tidak ditemukan",
			})
		case errors.Is(err, pppoe.ErrActiveConnectionNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Koneksi PPP aktif tidak ditemukan",
			})
		case errors.Is(err, pppoe.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "ID koneksi tidak valid",
			})
		default:
			c.JSON(http.StatusBadGateway, gin.H{
				"error": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Koneksi PPP aktif berhasil diputus",
	})
}
