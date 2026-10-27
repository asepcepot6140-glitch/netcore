package simplequeue

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/services/simplequeue"

	"github.com/gin-gonic/gin"
)

func getMikrotikID(c *gin.Context) (uint64, bool) {
	id, err := simplequeue.ParseMikrotikID(c.Query("mikrotik_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid mikrotik_id"})
		return 0, false
	}
	return id, true
}

// getItemID membaca :id dan memastikan formatnya ID RouterOS (*1A).
func getItemID(c *gin.Context) (string, bool) {
	id := strings.TrimSpace(c.Param("id"))
	if !simplequeue.ValidID(id) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
		return "", false
	}
	return id, true
}

// handleError mencatat detail di server; error koneksi router hanya dikirim sebagai pesan generik.
func handleError(c *gin.Context, err error) {
	var inputErr *simplequeue.InputError
	switch {
	case errors.As(err, &inputErr):
		c.JSON(http.StatusBadRequest, gin.H{"message": inputErr.Message})
	case errors.Is(err, simplequeue.ErrMikrotikNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "MikroTik not found"})
	case errors.Is(err, simplequeue.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "Simple Queue not found"})
	case errors.Is(err, simplequeue.ErrExists):
		c.JSON(http.StatusConflict, gin.H{"message": "Simple Queue name already exists"})
	// 409 (bukan 403) agar tidak dianggap sesi ditolak oleh interceptor auth di frontend.
	case errors.Is(err, simplequeue.ErrDynamic):
		c.JSON(http.StatusConflict, gin.H{"message": "Dynamic Simple Queue is managed by RouterOS and cannot be modified"})
	case errors.Is(err, simplequeue.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
	default:
		log.Printf("simple queue error: %s %s: %v", c.Request.Method, c.FullPath(), err)
		c.JSON(http.StatusBadGateway, gin.H{"message": "Failed to communicate with MikroTik"})
	}
}

// GET /api/queue/simple?mikrotik_id=1
func List(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	data, err := simplequeue.List(c.GetUint("user_id"), mikrotikID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// GET /api/queue/types?mikrotik_id=1
func ListTypes(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	data, err := simplequeue.ListTypeNames(c.GetUint("user_id"), mikrotikID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// GET /api/queue/parents?mikrotik_id=1
func ListParents(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	data, err := simplequeue.ListParentNames(c.GetUint("user_id"), mikrotikID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// GET /api/queue/packet-marks?mikrotik_id=1
func ListPacketMarks(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	data, err := simplequeue.ListPacketMarks(c.GetUint("user_id"), mikrotikID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// POST /api/queue/simple?mikrotik_id=1
func Create(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	var input simplequeue.Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid Simple Queue data"})
		return
	}

	if err := simplequeue.Create(c.GetUint("user_id"), mikrotikID, input); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Simple Queue added successfully"})
}

// PUT /api/queue/simple/:id?mikrotik_id=1
func Update(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	id, ok := getItemID(c)
	if !ok {
		return
	}

	var input simplequeue.Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid Simple Queue data"})
		return
	}

	if err := simplequeue.Update(c.GetUint("user_id"), mikrotikID, id, input); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Simple Queue updated successfully"})
}

// DELETE /api/queue/simple/:id?mikrotik_id=1
func Delete(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	id, ok := getItemID(c)
	if !ok {
		return
	}

	if err := simplequeue.Delete(c.GetUint("user_id"), mikrotikID, id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Simple Queue deleted successfully"})
}
