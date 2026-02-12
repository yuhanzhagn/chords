package controller

import (
	"backend/internal/model"
	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"net/http"
)

type UserController struct {
	Service service.UserService
}

func NewUserController(service service.UserService) *UserController {
	return &UserController{Service: service}
}

// CreateUser handles POST /users
func (c *UserController) CreateUser(ctx *gin.Context) {
	var user model.User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	if err := c.Service.CreateUser(&user); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	ctx.JSON(http.StatusCreated, user)
}

// GetUsers handles GET /users
func (c *UserController) GetUsers(ctx *gin.Context) {
	users, err := c.Service.GetAllUsers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	ctx.JSON(http.StatusOK, users)
}
