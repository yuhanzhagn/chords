package dto

import (
	"backend/internal/model"
	"time"
)

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
type UserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func User(u *model.User) UserResponse {
	return UserResponse{ID: u.ID, Username: u.Username, Email: u.Email, CreatedAt: u.CreatedAt}
}
