package hotspot

import (
	"net/http"

	"github.com/asepcepot6140-glitch/netcore/internal/services/hotspot"

	"github.com/gin-gonic/gin"
)

// GET /api/hotspot/profiles?mikrotik_id=1
func ListProfiles(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	data, err := hotspot.ListProfiles(c.GetUint("user_id"), mikrotikID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// GET /api/hotspot/profiles/:id?mikrotik_id=1
func ShowProfile(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	id, ok := getItemID(c)
	if !ok {
		return
	}

	data, err := hotspot.ShowProfile(c.GetUint("user_id"), mikrotikID, id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// POST /api/hotspot/profiles?mikrotik_id=1
func CreateProfile(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	var input hotspot.ProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid Hotspot Profile data"})
		return
	}

	if err := hotspot.CreateProfile(c.GetUint("user_id"), mikrotikID, input); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Hotspot Profile created successfully"})
}

// PUT /api/hotspot/profiles/:id?mikrotik_id=1
func UpdateProfile(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	id, ok := getItemID(c)
	if !ok {
		return
	}

	var input hotspot.ProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid Hotspot Profile data"})
		return
	}

	if err := hotspot.UpdateProfile(c.GetUint("user_id"), mikrotikID, id, input); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Hotspot Profile updated successfully"})
}

// DELETE /api/hotspot/profiles/:id?mikrotik_id=1
func DeleteProfile(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	id, ok := getItemID(c)
	if !ok {
		return
	}

	if err := hotspot.DeleteProfile(c.GetUint("user_id"), mikrotikID, id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Hotspot Profile deleted successfully"})
}
