package hotspot

import (
	"net/http"

	"github.com/asepcepot6140-glitch/netcore/internal/services/hotspot"

	"github.com/gin-gonic/gin"
)

// optionHandler membungkus fungsi daftar nama menjadi handler GET { data: []string }.
func optionHandler(list func(userID uint, mikrotikID uint64) ([]string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		mikrotikID, ok := getMikrotikID(c)
		if !ok {
			return
		}

		data, err := list(c.GetUint("user_id"), mikrotikID)
		if err != nil {
			handleError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": data})
	}
}

// GET /api/hotspot/pools?mikrotik_id=1
var ListPools = optionHandler(hotspot.ListPoolNames)

// GET /api/hotspot/queues?mikrotik_id=1
var ListQueues = optionHandler(hotspot.ListQueueNames)

// GET /api/hotspot/queue-types?mikrotik_id=1
var ListQueueTypes = optionHandler(hotspot.ListQueueTypeNames)
