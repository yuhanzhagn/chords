package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"backend/internal/dto"
	"backend/internal/model"
	"backend/internal/service"
)

type AuthController struct {
	Service service.AuthService
}

func NewAuthController(service service.AuthService) *AuthController {
	return &AuthController{Service: service}
}

func (c *AuthController) Register(ctx *gin.Context) {
	var body dto.CreateUserRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	user := model.User{Username: body.Username, Email: body.Email, Password: body.Password}
	if err := c.Service.Register(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Unable to register user"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Registered"})
}

func (c *AuthController) Login(ctx *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
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
