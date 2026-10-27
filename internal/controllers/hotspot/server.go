package hotspot

import (
	"net/http"

	"github.com/asepcepot6140-glitch/netcore/internal/services/hotspot"

	"github.com/gin-gonic/gin"
)

// GET /api/hotspot/servers?mikrotik_id=1
func ListServers(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	data, err := hotspot.ListServerNames(c.GetUint("user_id"), mikrotikID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}
