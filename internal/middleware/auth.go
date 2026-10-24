package middleware

import (
	"net/http"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/utils"
	"github.com/gin-gonic/gin"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		const bearerPrefix = "Bearer "

		authorization := c.GetHeader("Authorization")

		if len(authorization) <= len(bearerPrefix) ||
			!strings.EqualFold(authorization[:len(bearerPrefix)], bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
			})
			return
		}

		token := strings.TrimSpace(authorization[len(bearerPrefix):])
		claims, err := utils.ParseToken(token)

		if err != nil || claims.Type != "access" || claims.UserID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
