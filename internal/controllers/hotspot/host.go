package hotspot

import (
	"net/http"

	"github.com/asepcepot6140-glitch/netcore/internal/services/hotspot"

	"github.com/gin-gonic/gin"
)

// GET /api/hotspot/hosts?mikrotik_id=1
func ListHosts(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	data, err := hotspot.ListHosts(c.GetUint("user_id"), mikrotikID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// DELETE /api/hotspot/hosts/:id?mikrotik_id=1
func DeleteHost(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	id, ok := getItemID(c)
	if !ok {
		return
	}

	if err := hotspot.DeleteHost(c.GetUint("user_id"), mikrotikID, id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Hotspot Host deleted successfully"})
}
