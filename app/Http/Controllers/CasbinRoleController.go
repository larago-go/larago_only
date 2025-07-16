package Controllers

import (
	"larago/app/Model"
	"larago/config"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

func CasbinRole(router *gin.RouterGroup) {

	router.POST("/post_add", AddPostCasbinRole)
	router.GET("/api/add", ApiAddCasbinRole)
	router.GET("/api/list", ApiViewCasbinRole)
	router.DELETE("/api/list/:id/delete", ApiDeleteCasbinRole)

}

type CasbinRoleAddValidation struct {
	RoleName string `form:"rolename" json:"rolename" binding:"required"`
	Path     string `form:"path" json:"path" binding:"required"`
	Method   string `form:"method" json:"method" binding:"required"`
}

func AddPostCasbinRole(c *gin.Context) {
	// Validate input
	var input CasbinRoleAddValidation

	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role_c := Model.CasbinRoleConf{
		Ptype:    "p",
		RoleName: input.RoleName,
		Path:     input.Path,
		Method:   input.Method,
	}

	config.DB.Save(&role_c)

	// Create role
	role := Model.CasbinRoleModel{
		RoleName: input.RoleName,
		Path:     input.Path,
		Method:   input.Method,
	}

	//Gorm_SQL
	config.DB.Save(&role)

	c.IndentedJSON(http.StatusCreated, role)
}

func ApiViewCasbinRole(c *gin.Context) {

	//Gorm_SQL
	var model []Model.CasbinRoleModel

	claims, exists := c.Get("claims")

	if !exists {
		c.IndentedJSON(http.StatusOK, gin.H{"redirect": "redirect_auth_login"})
		c.Abort()
	}

	userClaims := claims.(*jwt.MapClaims)

	user_name := (*userClaims)["user_name"].(string)

	//Gorm_SQL
	config.DB.Find(&model)

	c.IndentedJSON(http.StatusOK, gin.H{
		"session_name": user_name,
		"list":         model,
	})

	//CasbinroleList.vue

}

func ApiAddCasbinRole(c *gin.Context) {

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

	//CasbinroleAdd.vue

}

func ApiDeleteCasbinRole(c *gin.Context) {

	var model Model.CasbinRoleModel
	var model_conf Model.CasbinRoleConf
	//Gorm_SQL
	if err := config.DB.Where("id = ?", c.Param("id")).First(&model).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Record not found!"})
		return
	}

	config.DB.Where("v0 = ?", model.RoleName).Where("v1 = ?", model.Path).Where("v2 = ?", model.Method).First(&model_conf)

	config.DB.Delete(&model_conf)

	config.DB.Delete(&model)

	c.IndentedJSON(http.StatusOK, gin.H{"data": true})
}
