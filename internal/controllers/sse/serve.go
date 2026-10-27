// Package sse berisi handler Server-Sent Events yang dipakai bersama oleh menu realtime.
package sse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/services/livestream"

	"github.com/gin-gonic/gin"
)

// writeTimeout memperbarui WriteTimeout server (60 detik, berlaku untuk seluruh respons) per kiriman.
const writeTimeout = 10 * time.Second

// Serve menjalankan run sebagai stream SSE: tiap emit menjadi satu event "data: <json>". Stream
// ditutup server tiap sekitar semenit dan klien menyambung lagi. Error sebelum event pertama
// dijawab sebagai HTTP biasa (429 bila slot stream habis, selain itu lewat onError); setelah
// stream berjalan header sudah terkirim, jadi error hanya dicatat.
func Serve[T any](
	c *gin.Context,
	name string,
	run func(ctx context.Context, emit func(T) error) error,
	onError func(*gin.Context, error),
) {
	controller := http.NewResponseController(c.Writer)
	emit := func(live T) error {
		if !c.Writer.Written() {
			header := c.Writer.Header()
			header.Set("Content-Type", "text/event-stream")
			header.Set("Cache-Control", "no-cache")
			header.Set("X-Accel-Buffering", "no") // nginx jangan menahan event
			c.Status(http.StatusOK)
		}
		payload, err := json.Marshal(live)
		if err != nil {
			return err
		}
		_ = controller.SetWriteDeadline(time.Now().Add(writeTimeout))
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", payload); err != nil {
			return err
		}
		return controller.Flush()
	}

	err := run(c.Request.Context(), emit)
	if err == nil {
		return
	}
	if c.Writer.Written() {
		log.Printf("%s live stream ended: %v", name, err)
		return
	}

	switch {
	case errors.Is(err, livestream.ErrTooManyStreams):
		tooMany(c, "Too many live connections. Close another tab and try again.")
	case errors.Is(err, livestream.ErrRouterStreamsFull):
		tooMany(c, "This MikroTik already has the maximum number of live connections. Try again shortly.")
	default:
		onError(c, err)
	}
}

// tooMany menjawab 429 dengan "message" (dipakai semua modul frontend).
func tooMany(c *gin.Context, message string) {
	c.JSON(http.StatusTooManyRequests, gin.H{"message": message})
}
