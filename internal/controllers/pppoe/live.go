package pppoe

import (
	"context"

	"github.com/asepcepot6140-glitch/netcore/internal/controllers/sse"
	"github.com/asepcepot6140-glitch/netcore/internal/services/pppoe"

	"github.com/gin-gonic/gin"
)

// GET /api/pppoe/live?mikrotik_id=1 (Server-Sent Events: secret, offline, dan active yang berubah tiap 3 detik)
func Live(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	userID := c.GetUint("user_id")
	sse.Serve(c, "pppoe", func(ctx context.Context, emit func(pppoe.Live) error) error {
		return pppoe.StreamLive(ctx, userID, mikrotikID, emit)
	}, handleError)
}
