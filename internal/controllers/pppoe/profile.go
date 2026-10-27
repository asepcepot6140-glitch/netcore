package pppoe

import (
	"errors"
	"net/http"

	"github.com/asepcepot6140-glitch/netcore/internal/services/pppoe"

	"github.com/gin-gonic/gin"
)

func handleProfileError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, pppoe.ErrMikrotikNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "MikroTik not found"})
	case errors.Is(err, pppoe.ErrProfileNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "PPP Profile not found"})
	case errors.Is(err, pppoe.ErrNameExists):
		c.JSON(http.StatusConflict, gin.H{"error": "PPP Profile name already exists"})
	case errors.Is(err, pppoe.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
	default:
		upstreamError(c, err)
	}
}

func getProfileParams(c *gin.Context) (uint64, bool) {
	mikrotikID, err := pppoe.ParseMikrotikID(c.Query("mikrotik_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mikrotik_id"})
		return 0, false
	}

	return mikrotikID, true
}

// GET /api/pppoe/profiles?mikrotik_id=1
func ListProfiles(c *gin.Context) {
	mikrotikID, ok := getProfileParams(c)
	if !ok {
		return
	}

	profiles, err := pppoe.ListProfiles(c.GetUint("user_id"), mikrotikID)
	if err != nil {
		handleProfileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": profiles})
}

// GET /api/pppoe/profiles/:name?mikrotik_id=1
func ShowProfile(c *gin.Context) {
	mikrotikID, ok := getProfileParams(c)
	if !ok {
		return
	}

	profile, err := pppoe.ShowProfile(
		c.GetUint("user_id"),
		mikrotikID,
		c.Param("name"),
	)
	if err != nil {
		handleProfileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": profile})
}

// POST /api/pppoe/profiles
func CreateProfile(c *gin.Context) {
	mikrotikID, ok := getProfileParams(c)
	if !ok {
		return
	}

	var input pppoe.ProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	if err := pppoe.CreateProfile(
		c.GetUint("user_id"),
		mikrotikID,
		input,
	); err != nil {
		handleProfileError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "PPP Profile created successfully"})
}

// PUT /api/pppoe/profiles/:name?mikrotik_id=1
func UpdateProfile(c *gin.Context) {
	mikrotikID, ok := getProfileParams(c)
	if !ok {
		return
	}

	var input pppoe.ProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	if err := pppoe.UpdateProfile(
		c.GetUint("user_id"),
		mikrotikID,
		c.Param("name"),
		input,
	); err != nil {
		handleProfileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "PPP Profile updated successfully"})
}

// DELETE /api/pppoe/profiles/:name?mikrotik_id=1
func DeleteProfile(c *gin.Context) {
	mikrotikID, ok := getProfileParams(c)
	if !ok {
		return
	}

	if err := pppoe.DeleteProfile(
		c.GetUint("user_id"),
		mikrotikID,
		c.Param("name"),
	); err != nil {
		handleProfileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "PPP Profile deleted successfully"})
}
