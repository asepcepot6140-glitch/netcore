package pppoe

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// upstreamError mencatat detail error di server dan hanya mengirim pesan generik ke client.
func upstreamError(c *gin.Context, err error) {
	log.Printf("pppoe upstream error: %s %s: %v", c.Request.Method, c.FullPath(), err)

	c.JSON(http.StatusBadGateway, gin.H{
		"error": "Failed to communicate with MikroTik",
	})
}
