package controller

import (
	"backend/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MessageController struct {
	MessageService service.MessageService
}

// NewMessageController creates a new controller
func NewMessageController(ms service.MessageService) *MessageController {
	return &MessageController{MessageService: ms}
}

// POST /messages
func (mc *MessageController) CreateMessage(c *gin.Context) {
	var input struct {
		Content    string `json:"content" binding:"required"`
		UserID     uint   `json:"user_id" binding:"required"`
		ChatRoomID uint   `json:"chat_room_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := mc.MessageService.CreateMessage(input.UserID, input.ChatRoomID, input.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

// GET /chatrooms/:id/messages
func (mc *MessageController) GetMessagesByChatRoom(c *gin.Context) {
	chatRoomIDParam := c.Param("id")
	chatRoomID, err := strconv.ParseUint(chatRoomIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat room id"})
		return
	}

	messages, err := mc.MessageService.GetMessagesByChatRoom(uint(chatRoomID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// DELETE /messages/:id
func (mc *MessageController) DeleteMessage(c *gin.Context) {
	idParam := c.Param("id")
	msgID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	if err := mc.MessageService.DeleteMessage(uint(msgID)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.AbortWithStatus(http.StatusNoContent)
}
