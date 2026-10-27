package hotspot

import (
	"context"

	"github.com/asepcepot6140-glitch/netcore/internal/controllers/sse"
	"github.com/asepcepot6140-glitch/netcore/internal/services/hotspot"

	"github.com/gin-gonic/gin"
)

// GET /api/hotspot/live?mikrotik_id=1 (Server-Sent Events: user, active, dan host yang berubah tiap 3 detik)
func Live(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	userID := c.GetUint("user_id")
	sse.Serve(c, "hotspot", func(ctx context.Context, emit func(hotspot.Live) error) error {
		return hotspot.StreamLive(ctx, userID, mikrotikID, emit)
	}, handleError)
}
