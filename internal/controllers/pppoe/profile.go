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
		c.JSON(http.StatusNotFound, gin.H{"error": "MikroTik tidak ditemukan"})
	case errors.Is(err, pppoe.ErrProfileNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "PPP Profile tidak ditemukan"})
	case errors.Is(err, pppoe.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Input tidak valid"})
	default:
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	}
}

func getProfileParams(c *gin.Context) (uint64, bool) {
	mikrotikID, err := pppoe.ParseMikrotikID(c.Query("mikrotik_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mikrotik_id tidak valid"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format JSON tidak valid"})
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

	c.JSON(http.StatusCreated, gin.H{"message": "PPP Profile berhasil dibuat"})
}

// PUT /api/pppoe/profiles/:name?mikrotik_id=1
func UpdateProfile(c *gin.Context) {
	mikrotikID, ok := getProfileParams(c)
	if !ok {
		return
	}

	var input pppoe.ProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format JSON tidak valid"})
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

	c.JSON(http.StatusOK, gin.H{"message": "PPP Profile berhasil diperbarui"})
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

	c.JSON(http.StatusOK, gin.H{"message": "PPP Profile berhasil dihapus"})
}
