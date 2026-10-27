package iface

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/services/iface"

	"github.com/gin-gonic/gin"
)

func getMikrotikID(c *gin.Context) (uint64, bool) {
	id, err := iface.ParseMikrotikID(c.Query("mikrotik_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid mikrotik_id"})
		return 0, false
	}
	return id, true
}

// getItemID membaca :id dan memastikan formatnya ID RouterOS (*1A).
func getItemID(c *gin.Context) (string, bool) {
	id := strings.TrimSpace(c.Param("id"))
	if !iface.ValidID(id) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
		return "", false
	}
	return id, true
}

// handleError mencatat detail di server; error koneksi router hanya dikirim sebagai pesan generik.
func handleError(c *gin.Context, err error) {
	var inputErr *iface.InputError
	switch {
	case errors.As(err, &inputErr):
		c.JSON(http.StatusBadRequest, gin.H{"message": inputErr.Message})
	case errors.Is(err, iface.ErrMikrotikNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "MikroTik not found"})
	case errors.Is(err, iface.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "Interface not found"})
	case errors.Is(err, iface.ErrExists):
		c.JSON(http.StatusConflict, gin.H{"message": "Interface name already exists"})
	// 409 (bukan 403) agar tidak dianggap sesi ditolak oleh interceptor auth di frontend.
	case errors.Is(err, iface.ErrDynamic):
		c.JSON(http.StatusConflict, gin.H{"message": "Dynamic interface is managed by RouterOS and cannot be modified"})
	case errors.Is(err, iface.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
	case errors.Is(err, iface.ErrTooManyStreams):
		c.JSON(http.StatusTooManyRequests, gin.H{"message": "Too many live connections. Close another Interface tab and try again."})
	case errors.Is(err, iface.ErrRouterStreamsFull):
		c.JSON(http.StatusTooManyRequests, gin.H{"message": "This MikroTik already has the maximum number of live connections. Try again shortly."})
	default:
		log.Printf("interface error: %s %s: %v", c.Request.Method, c.FullPath(), err)
		c.JSON(http.StatusBadGateway, gin.H{"message": "Failed to communicate with MikroTik"})
	}
}

// GET /api/interfaces?mikrotik_id=1
func List(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	data, err := iface.List(c.GetUint("user_id"), mikrotikID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// GET /api/interfaces/live?mikrotik_id=1 (Server-Sent Events: rate tiap detik, status tiap 2 detik)
// Server-Sent Events: satu event per detik berisi rate interface yang berubah. Stream ditutup
// server setiap sekitar semenit; klien menyambung lagi.
func Live(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	controller := http.NewResponseController(c.Writer)
	emit := func(live iface.Live) error {
		if !c.Writer.Written() {
			header := c.Writer.Header()
			header.Set("Content-Type", "text/event-stream")
			header.Set("Cache-Control", "no-cache")
			header.Set("X-Accel-Buffering", "no") // nginx jangan menahan event
			c.Status(http.StatusOK)
		}
		// nil harus jadi [], bukan null.
		live.Rates = append([]iface.Rate{}, live.Rates...)
		live.Status = append([]iface.Status{}, live.Status...)
		payload, err := json.Marshal(live)
		if err != nil {
			return err
		}
		// WriteTimeout server (60 detik) berlaku untuk seluruh respons; stream memperbaruinya per kiriman.
		_ = controller.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", payload); err != nil {
			return err
		}
		return controller.Flush()
	}

	err := iface.StreamLive(c.Request.Context(), c.GetUint("user_id"), mikrotikID, emit)
	if err == nil {
		return
	}
	// Setelah stream berjalan, header sudah terkirim; klien cukup menyambung ulang.
	if c.Writer.Written() {
		log.Printf("interface live stream ended: %v", err)
		return
	}
	handleError(c, err)
}

// GET /api/interfaces/:id/monitor?mikrotik_id=1
func Monitor(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	id, ok := getItemID(c)
	if !ok {
		return
	}

	data, err := iface.Monitor(c.GetUint("user_id"), mikrotikID, id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// PUT /api/interfaces/:id?mikrotik_id=1
func Update(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	id, ok := getItemID(c)
	if !ok {
		return
	}

	var input iface.Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid interface data"})
		return
	}

	if err := iface.Update(c.GetUint("user_id"), mikrotikID, id, input); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Interface updated successfully"})
}
