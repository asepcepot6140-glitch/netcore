package hotspot

import (
	"net/http"

	"github.com/asepcepot6140-glitch/netcore/internal/services/hotspot"

	"github.com/gin-gonic/gin"
)

// GET /api/hotspot/active?mikrotik_id=1
func ListActive(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	data, err := hotspot.ListActive(c.GetUint("user_id"), mikrotikID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// DELETE /api/hotspot/active/:id?mikrotik_id=1
func KickActive(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	id, ok := getItemID(c)
	if !ok {
		return
	}

	if err := hotspot.KickActive(c.GetUint("user_id"), mikrotikID, id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Hotspot session disconnected successfully"})
}
