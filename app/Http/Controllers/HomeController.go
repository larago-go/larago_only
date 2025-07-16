package Controllers

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"

	"net/http"
)

func Home(router *gin.RouterGroup) {

	router.GET("/api", ApiViewHome)

}

func ApiViewHome(c *gin.Context) {

	claims, exists := c.Get("claims")

	if !exists {
		c.IndentedJSON(http.StatusOK, gin.H{"redirect": "redirect_auth_login"})
		c.Abort()
	}

	userClaims := claims.(*jwt.MapClaims)

	user_name := (*userClaims)["user_name"].(string)

	c.IndentedJSON(http.StatusOK, gin.H{
		"session_name": user_name,
	})

	// DashboardAdmin.vue

}
