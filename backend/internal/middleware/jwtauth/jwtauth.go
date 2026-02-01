package jwtauth

import (
	"net/http"
    "strings"

    "github.com/gin-gonic/gin"
	"backend/utils"
	"backend/internal/service"
)

func JWTAuthMiddleware(authSvc service.AuthService) gin.HandlerFunc {
    return func(c *gin.Context) {
        auth := c.GetHeader("Authorization")
        if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
            return
        }

        tokenStr := strings.TrimPrefix(auth, "Bearer ")

        claims, err := utils.ParseJWT(tokenStr)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }

        // JTI check
        if authSvc.IsBlocked(claims.ID) {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
            return
        }

        // Attach claims to context
        c.Set("uid", claims.Username)
        c.Set("jti", claims.ID)
        c.Set("exp", claims.ExpiresAt.Time)

        c.Next()
    }
}

