package middleware

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func MWtoken() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("X-DMP-TOKEN")
		claims, err := utils.ValidateJWT(token, []byte(db.JwtSecret))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 420, "message": utils.I18n.Get(c, "token fail"), "data": nil})
			c.Abort()
			return
		}
		c.Set("username", claims.Username)
		c.Set("nickname", claims.Nickname)
		c.Set("role", claims.Role)
		c.Next()
	}
}
