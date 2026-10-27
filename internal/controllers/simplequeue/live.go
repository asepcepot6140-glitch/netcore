package simplequeue

import (
	"context"

	"github.com/asepcepot6140-glitch/netcore/internal/controllers/sse"
	"github.com/asepcepot6140-glitch/netcore/internal/services/simplequeue"

	"github.com/gin-gonic/gin"
)

// GET /api/queue/simple/live?mikrotik_id=1 (Server-Sent Events: queue yang berubah tiap 3 detik)
func Live(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	userID := c.GetUint("user_id")
	sse.Serve(c, "simple queue", func(ctx context.Context, emit func(simplequeue.Live) error) error {
		return simplequeue.StreamLive(ctx, userID, mikrotikID, emit)
	}, handleError)
}
