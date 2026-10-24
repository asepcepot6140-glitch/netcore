package genieacs

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/asepcepot6140-glitch/netcore/internal/services/genieacs"

	"github.com/gin-gonic/gin"
)

type GenieACSRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
	Host string `json:"host" binding:"required,max=255"`
	Port int    `json:"port" binding:"required,min=1,max=65535"`
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid GenieACS ID",
		})
		return 0, false
	}

	return uint(id), true
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, genieacs.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"message": "GenieACS configuration not found",
		})

	case errors.Is(err, genieacs.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid GenieACS data",
		})

	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Internal server error",
		})
	}
}

func List(c *gin.Context) {
	userID := c.GetUint("user_id")

	items, err := genieacs.List(userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
	})
}

func Show(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	userID := c.GetUint("user_id")

	item, err := genieacs.Show(userID, id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": item,
	})
}

func Create(c *gin.Context) {
	var req GenieACSRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid GenieACS data",
		})
		return
	}

	userID := c.GetUint("user_id")

	item, err := genieacs.Create(
		userID,
		genieacs.GenieACSInput{
			Name: req.Name,
			Host: req.Host,
			Port: req.Port,
		},
	)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "GenieACS added successfully",
		"data":    item,
	})
}

func Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var req GenieACSRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid GenieACS data",
		})
		return
	}

	userID := c.GetUint("user_id")

	item, err := genieacs.Update(
		userID,
		id,
		genieacs.GenieACSInput{
			Name: req.Name,
			Host: req.Host,
			Port: req.Port,
		},
	)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "GenieACS updated successfully",
		"data":    item,
	})
}

func Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	userID := c.GetUint("user_id")

	if err := genieacs.Delete(userID, id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "GenieACS deleted successfully",
	})
}
