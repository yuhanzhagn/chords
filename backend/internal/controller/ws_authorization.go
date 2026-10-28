package controller

import (
	"backend/internal/identity"
	"backend/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type WSAuthorizer interface {
	AuthorizeWS(identity.Principal, uint, string, string) (service.WSGrant, error)
}

func WSAuthorize(authorizer WSAuthorizer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			RoomID uint   `json:"room_id"`
			Kind   string `json:"kind"`
			Digest string `json:"digest"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": "Invalid input"})
			return
		}
		grant, err := authorizer.AuthorizeWS(identity.FromContext(c.Request.Context()), input.RoomID, input.Kind, input.Digest)
		if err != nil {
			c.JSON(serviceStatus(err, 500), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, grant)
	}
}
