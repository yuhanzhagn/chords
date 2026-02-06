package controller

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"backend/internal/model"
	"backend/internal/service"
	"backend/utils"
)

type AuthController struct {
	Service service.AuthService
}

func NewAuthController(service service.AuthService) *AuthController {
	return &AuthController{Service: service}
}

func (c *AuthController) Register(ctx *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	hash, _ := utils.HashPassword(body.Password)

	user := model.User{Username: body.Username, Email: body.Email, Password: hash}
	if err := c.Service.Register(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Registered"})
}

func (c *AuthController) Login(ctx *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	id, err := c.Service.GetUserIDByUsername(body.Username)
	if err != nil {
		log.Println(err)
	}

	//force logout other account and stop their websocket
	err = c.Service.ForceLogoutAll(id)
	if err != nil {
		log.Println(err)
	}

	id, token, jti, sessionID, err := c.Service.Login(body.Username, body.Password)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"id": id, "token": token, "jti": jti, "sessionID": sessionID})
}

// TODO move that logout function into service
func (c *AuthController) Logout(ctx *gin.Context) {
	// Get Authorization header
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing Authorization header"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	err := c.Service.Logout(tokenString)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "logout successful",
	})
}
