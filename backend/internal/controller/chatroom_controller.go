package controller

import (
	"net/http"
	"strconv"

	"backend/internal/service"
	"github.com/gin-gonic/gin"
)

type ChatRoomController struct {
	chatRoomService service.ChatRoomService
}

func NewChatRoomController(chatRoomService service.ChatRoomService) *ChatRoomController {
	return &ChatRoomController{chatRoomService: chatRoomService}
}

// POST /chatrooms
func (c *ChatRoomController) CreateChatRoom(ctx *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chatRoom, err := c.chatRoomService.CreateChatRoom(req.Name)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": chatRoom})
}

// GET /chatrooms
func (c *ChatRoomController) GetAllChatRooms(ctx *gin.Context) {
	chatRooms, err := c.chatRoomService.GetAllChatRooms()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": chatRooms})
}

// GET /chatrooms/:id
func (c *ChatRoomController) GetChatRoomByID(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	chatRoom, err := c.chatRoomService.GetChatRoomByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": chatRoom})
}

// DELETE /chatrooms/:id
func (c *ChatRoomController) DeleteChatRoom(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := c.chatRoomService.DeleteChatRoom(uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "chat room deleted"})
}

// GET /chatrooms/search?q=keyword
func (c *ChatRoomController) SearchChatRooms(ctx *gin.Context) {
	keyword := ctx.Query("q")
	if keyword == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "query keyword required"})
		return
	}

	chatRooms, err := c.chatRoomService.SearchChatRoomsByName(keyword)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": chatRooms})
}
