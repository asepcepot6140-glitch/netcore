package genieacs

import (
	"net/http"
	"strconv"

	genieacsService "github.com/asepcepot6140-glitch/netcore/internal/services/genieacs"

	"github.com/gin-gonic/gin"
)

func ListDevices(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "ID GenieACS tidak valid",
		})
		return
	}

	userID := c.GetUint("user_id")
	item, err := genieacsService.Show(userID, uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Konfigurasi GenieACS tidak ditemukan",
		})
		return
	}

	devices, err := genieacsService.GetDevices(c.Request.Context(), item)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "Gagal mengambil perangkat dari GenieACS",
		})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "ID GenieACS tidak valid",
		})
		return
	}

	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "ID device tidak boleh kosong",
		})
		return
	}

	userID := c.GetUint("user_id")
	item, err := genieacsService.Show(userID, uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Konfigurasi GenieACS tidak ditemukan",
		})
		return
	}

	err = genieacsService.SummonDevice(
		c.Request.Context(),
		item,
		deviceID,
	)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Permintaan refresh object berhasil dikirim",
	})
}
