package hotspot

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/services/hotspot"

	"github.com/gin-gonic/gin"
)

func getMikrotikID(c *gin.Context) (uint64, bool) {
	id, err := hotspot.ParseMikrotikID(c.Query("mikrotik_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid mikrotik_id"})
		return 0, false
	}
	return id, true
}

// getItemID membaca :id dan memastikan formatnya ID RouterOS (*1A).
func getItemID(c *gin.Context) (string, bool) {
	id := strings.TrimSpace(c.Param("id"))
	if !hotspot.ValidID(id) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
		return "", false
	}
	return id, true
}

// handleError mencatat detail di server dan hanya mengirim pesan generik untuk error router.
func handleError(c *gin.Context, err error) {
	var inputErr *hotspot.InputError
	switch {
	case errors.As(err, &inputErr):
		c.JSON(http.StatusBadRequest, gin.H{"message": inputErr.Message})
	case errors.Is(err, hotspot.ErrMikrotikNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "MikroTik not found"})
	case errors.Is(err, hotspot.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "Hotspot User not found"})
	case errors.Is(err, hotspot.ErrProfileNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "Hotspot Profile not found"})
	case errors.Is(err, hotspot.ErrActiveNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "Hotspot Active session not found"})
	case errors.Is(err, hotspot.ErrHostNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "Hotspot Host not found"})
	case errors.Is(err, hotspot.ErrUserExists):
		c.JSON(http.StatusConflict, gin.H{"message": "Hotspot User name already exists"})
	case errors.Is(err, hotspot.ErrProfileExists):
		c.JSON(http.StatusConflict, gin.H{"message": "Hotspot Profile name already exists"})
	case errors.Is(err, hotspot.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
	default:
		log.Printf("hotspot error: %s %s: %v", c.Request.Method, c.FullPath(), err)
		c.JSON(http.StatusBadGateway, gin.H{"message": "Failed to communicate with MikroTik"})
	}
}
