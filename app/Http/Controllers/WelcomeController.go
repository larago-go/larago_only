package Controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Welcome(router *gin.RouterGroup) {

	router.GET("/", GetWelcome)
	router.GET("/welcome", GetWelcome)

}

func GetWelcome(c *gin.Context) {

	c.IndentedJSON(http.StatusOK, gin.H{
		"larago": "ok",
	})

}
