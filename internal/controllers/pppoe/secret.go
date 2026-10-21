package pppoe

import (
	"errors"
	"net/http"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/services/pppoe"

	"github.com/gin-gonic/gin"
)

func getMikrotikID(c *gin.Context) (uint64, bool) {
	id, err := pppoe.ParseMikrotikID(c.Query("mikrotik_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "mikrotik_id tidak valid"})
		return 0, false
	}
	return id, true
}

func getSecretID(c *gin.Context) (string, bool) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" || !strings.HasPrefix(id, "*") {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID PPP Secret tidak valid"})
		return "", false
	}
	return id, true
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, pppoe.ErrMikrotikNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "MikroTik tidak ditemukan"})
	case errors.Is(err, pppoe.ErrSecretNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "PPP Secret tidak ditemukan"})
	case errors.Is(err, pppoe.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	default:
		c.JSON(http.StatusBadGateway, gin.H{"message": "Gagal berkomunikasi dengan MikroTik"})
	}
}

func ListSecrets(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	data, err := pppoe.ListSecrets(c.GetUint("user_id"), mikrotikID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func ShowSecret(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	secretID, ok := getSecretID(c)
	if !ok {
		return
	}

	data, err := pppoe.ShowSecret(c.GetUint("user_id"), mikrotikID, secretID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func CreateSecret(c *gin.Context) {
	var input pppoe.SecretInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Data PPP Secret tidak valid"})
		return
	}

	mikrotikID, err := pppoe.ParseMikrotikID(c.Query("mikrotik_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "mikrotik_id tidak valid"})
		return
	}

	if err := pppoe.CreateSecret(c.GetUint("user_id"), mikrotikID, input); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "PPP Secret berhasil ditambahkan"})
}

func UpdateSecret(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	secretID, ok := getSecretID(c)
	if !ok {
		return
	}

	var input pppoe.SecretInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Data PPP Secret tidak valid"})
		return
	}

	if err := pppoe.UpdateSecret(c.GetUint("user_id"), mikrotikID, secretID, input); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "PPP Secret berhasil diperbarui"})
}

func DeleteSecret(c *gin.Context) {
	mikrotikID, ok := getMikrotikID(c)
	if !ok {
		return
	}

	secretID, ok := getSecretID(c)
	if !ok {
		return
	}

	if err := pppoe.DeleteSecret(c.GetUint("user_id"), mikrotikID, secretID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "PPP Secret berhasil dihapus"})
}
