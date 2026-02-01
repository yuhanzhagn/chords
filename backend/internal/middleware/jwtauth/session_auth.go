package jwtauth

import (
    "net/http"
    "strings"
//    "time"

    "github.com/gin-gonic/gin"
  //  "backend/internal/model"
	"backend/internal/service"
    //"backend/utils"
    //"gorm.io/gorm"
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
		_, _, err := m.AuthSvc.ValidateJWT(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
            return
		}	
      
        c.Next()
    }
}

