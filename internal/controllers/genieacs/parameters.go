package genieacs

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/services/genieacs"
	"github.com/gin-gonic/gin"
)

const maxParametersBodyBytes = 32 << 10

type updateParametersRequest struct {
	Path   string            `json:"path"`
	Values map[string]string `json:"values" binding:"required,min=1,max=20"`
}

func updateParameters(target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}

		deviceID := c.Param("deviceId")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Device ID is required"})
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxParametersBodyBytes)

		var req updateParametersRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid parameter data"})
			return
		}

		item, err := genieacs.Show(c.GetUint("user_id"), id)
		if err != nil {
			handleError(c, err)
			return
		}

		applied, err := genieacs.SetParameters(
			c.Request.Context(),
			item,
			deviceID,
			target,
			req.Path,
			req.Values,
		)
		switch {
		case err == nil:
		case errors.Is(err, genieacs.ErrInvalidParameter),
			errors.Is(err, genieacs.ErrParameterNotSupported):
			message := err.Error()
			c.JSON(http.StatusBadRequest, gin.H{
				"message": strings.ToUpper(message[:1]) + message[1:],
			})
			return
		case errors.Is(err, genieacs.ErrDeviceRejected):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": rejectedMessage(err)})
			return
		case errors.Is(err, genieacs.ErrTargetNotFound):
			c.JSON(http.StatusNotFound, gin.H{"message": "Target object not found on device"})
			return
		default:
			log.Printf("genieacs set parameters error: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"message": "Failed to send changes to GenieACS"})
			return
		}

		message := "Changes applied successfully"
		if !applied {
			message = "Changes queued. The device has not confirmed them yet."
		}
		c.JSON(http.StatusOK, gin.H{"message": message, "applied": applied})
	}
}

// rejectedMessage turns a device fault into a client-facing message.
func rejectedMessage(err error) string {
	detail := strings.TrimPrefix(err.Error(), genieacs.ErrDeviceRejected.Error()+": ")
	return "The device rejected the change (" + detail + "). Nothing was changed."
}

type deleteWANRequest struct {
	Path string `json:"path" binding:"required"`
}

func DeleteWAN(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Device ID is required"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxParametersBodyBytes)

	var req deleteWANRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid WAN instance path"})
		return
	}

	item, err := genieacs.Show(c.GetUint("user_id"), id)
	if err != nil {
		handleError(c, err)
		return
	}

	applied, err := genieacs.DeleteWANConnection(c.Request.Context(), item, deviceID, req.Path)
	switch {
	case err == nil:
	case errors.Is(err, genieacs.ErrInvalidParameter):
		message := err.Error()
		c.JSON(http.StatusBadRequest, gin.H{
			"message": strings.ToUpper(message[:1]) + message[1:],
		})
		return
	case errors.Is(err, genieacs.ErrDeviceRejected):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": rejectedMessage(err)})
		return
	case errors.Is(err, genieacs.ErrTargetNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "Target object not found on device"})
		return
	default:
		log.Printf("genieacs delete wan error: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"message": "Failed to send delete request to GenieACS"})
		return
	}

	message := "WAN instance deleted successfully"
	if !applied {
		message = "Delete queued. The device has not confirmed it yet."
	}
	c.JSON(http.StatusOK, gin.H{"message": message, "applied": applied})
}

type addWANRequest struct {
	Kind string `json:"kind" binding:"required"`
}

func AddWAN(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Device ID is required"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxParametersBodyBytes)

	var req addWANRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid WAN type"})
		return
	}

	item, err := genieacs.Show(c.GetUint("user_id"), id)
	if err != nil {
		handleError(c, err)
		return
	}

	applied, err := genieacs.AddWANConnection(c.Request.Context(), item, deviceID, req.Kind)
	switch {
	case err == nil:
	case errors.Is(err, genieacs.ErrInvalidParameter):
		message := err.Error()
		c.JSON(http.StatusBadRequest, gin.H{
			"message": strings.ToUpper(message[:1]) + message[1:],
		})
		return
	case errors.Is(err, genieacs.ErrDeviceRejected):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": rejectedMessage(err)})
		return
	case errors.Is(err, genieacs.ErrTargetNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "WAN device not found on device"})
		return
	default:
		log.Printf("genieacs add wan error: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"message": "Failed to send add request to GenieACS"})
		return
	}

	message := "WAN instance added successfully"
	if !applied {
		message = "Add queued. The device has not confirmed it yet."
	}
	c.JSON(http.StatusOK, gin.H{"message": message, "applied": applied})
}

var (
	UpdateWAN     = updateParameters("wan")
	UpdateLANDHCP = updateParameters("lan/dhcp")
	UpdateLANPort = updateParameters("lan/ports")
	UpdateWLAN    = updateParameters("wlan")
)
