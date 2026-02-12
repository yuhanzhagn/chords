package controller

import (
	"backend/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MembershipController struct {
	membershipService service.MembershipService
}

func NewMembershipController(m service.MembershipService) *MembershipController {
	return &MembershipController{m}
}

// Request body now uses "username"
type UserChatroom struct {
	Username   string `json:"username" binding:"required"`
	ChatRoomID uint   `json:"chatroomid" binding:"required"`
}

func (c *MembershipController) AddUser(ctx *gin.Context) {
	var req UserChatroom
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := c.membershipService.AddUserToChatRoom(req.Username, req.ChatRoomID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":     "user added to chatroom",
		"username":    req.Username,
		"chatroom_id": req.ChatRoomID,
	})
}

func (c *MembershipController) GetUserChatRooms(ctx *gin.Context, username string) {

	rooms, err := c.membershipService.GetUserSubscribedChatRooms(username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": rooms})

}
