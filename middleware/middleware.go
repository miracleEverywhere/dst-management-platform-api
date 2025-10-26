package middleware

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func message(c *gin.Context, message string) string {
	zh := map[string]string{
		"token": "Token认证失败",
	}
	en := map[string]string{
		"token": "Token Auth Fail",
	}

	switch c.Request.Header.Get("X-I18n-Lang") {
	case "zh":
		return zh[message]
	case "en":
		return en[message]
	default:
		return zh[message]
	}
}

func MWtoken() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("X-DMP-TOKEN")
		claims, err := utils.ValidateJWT(token, []byte(db.JwtSecret))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 420, "message": message(c, "token"), "data": nil})
			c.Abort()
			return
		}
		c.Set("username", claims.Username)
		c.Set("nickname", claims.Nickname)
		c.Set("role", claims.Role)
		c.Next()
	}
}
