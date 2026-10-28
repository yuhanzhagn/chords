package controller

import (
	"backend/internal/dto"
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
	var body dto.CreateUserRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	user := model.User{Username: body.Username, Email: body.Email, Password: body.Password}
	if err := c.Service.CreateUser(&user); err != nil {
		ctx.JSON(serviceStatus(err, http.StatusInternalServerError), gin.H{"error": "Failed to create user"})
		return
	}

	ctx.JSON(http.StatusCreated, dto.User(&user))
}

// GetUsers handles GET /users
func (c *UserController) GetUsers(ctx *gin.Context) {
	users, err := c.Service.GetAllUsers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	responses := make([]dto.UserResponse, 0, len(users))
	for i := range users {
		responses = append(responses, dto.User(&users[i]))
	}
	ctx.JSON(http.StatusOK, responses)
}
