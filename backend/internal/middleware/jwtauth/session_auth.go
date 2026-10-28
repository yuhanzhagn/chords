package jwtauth

import (
	"backend/internal/identity"
	"errors"
	"net/http"
	"strings"

	"backend/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	AuthSvc service.AuthService
}

func NewAuthMiddleware(authSvc service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{AuthSvc: authSvc}
}

// Middleware function
func (m *AuthMiddleware) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid Authorization header"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		user, session, err := m.AuthSvc.ValidateJWT(tokenString)
		if errors.Is(err, service.ErrAuthenticationUnavailable) {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Authentication temporarily unavailable"})
			return
		}
		if err != nil || user == nil || session == nil || session.UserID != user.ID {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
			return
		}

		p := identity.New(user.ID, user.Username, session.SessionID)
		if !p.Valid() {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Request = c.Request.WithContext(identity.WithContext(c.Request.Context(), p))
		c.Next()
	}
}
