package controller

import (
	"backend/internal/identity"
	"backend/internal/service"
	kafkapb "backend/proto/kafka"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
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
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, kafkapb.MaxRESTBodyBytes)
	var input struct {
		RequestID  string `json:"request_id"`
		TempID     string `json:"temp_id"`
		Content    string `json:"content" binding:"required"`
		UserID     uint   `json:"user_id" binding:"required"`
		ChatRoomID uint   `json:"chat_room_id" binding:"required"`
	}

	if err := jsonBindMessage(c, &input); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "message too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := mc.MessageService.SendMessage(identity.FromContext(c.Request.Context()), service.SendInput{UserID: input.UserID, RoomID: input.ChatRoomID, Content: input.Content, RequestID: input.RequestID, TempID: input.TempID})
	if err != nil {
		c.JSON(serviceStatus(err, http.StatusInternalServerError), gin.H{"error": err.Error()})
		return
	}

	status := http.StatusCreated
	if msg.Replayed {
		status = http.StatusOK
	}
	c.JSON(status, msg)
}

// GET /chatrooms/:id/messages
func (mc *MessageController) GetMessagesByChatRoom(c *gin.Context) {
	chatRoomIDParam := c.Param("id")
	chatRoomID, err := strconv.ParseUint(chatRoomIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat room id"})
		return
	}

	if chatRoomID == 0 || chatRoomID > math.MaxUint32 {
		c.JSON(400, gin.H{"error": "invalid room"})
		return
	}
	query := c.Request.URL.Query()
	for _, name := range []string{"before_seq", "after_seq", "snapshot_seq", "before_id", "limit"} {
		if values, ok := query[name]; ok && (len(values) != 1 || values[0] == "") {
			c.JSON(400, gin.H{"error": "invalid cursor or limit"})
			return
		}
	}

	if c.Query("before_seq") != "" || c.Query("after_seq") != "" {
		var cursor service.HistoryCursor
		cursor.Limit = 30
		if raw, ok := c.GetQuery("limit"); ok {
			n, e := strconv.Atoi(raw)
			if e != nil || n < 1 || n > 100 {
				c.JSON(400, gin.H{"error": "invalid limit"})
				return
			}
			cursor.Limit = n
		}
		for name, target := range map[string]**uint64{"before_seq": &cursor.Before, "after_seq": &cursor.After, "snapshot_seq": &cursor.Snapshot} {
			if raw, ok := c.GetQuery(name); ok {
				n, e := strconv.ParseUint(raw, 10, 63)
				if e != nil || raw == "" {
					c.JSON(400, gin.H{"error": "invalid cursor"})
					return
				}
				*target = &n
			}
		}
		if _, ok := c.GetQuery("before_id"); ok {
			c.JSON(400, gin.H{"error": "mixed cursors"})
			return
		}
		history, ok := mc.MessageService.(service.SequenceHistory)
		if !ok {
			c.JSON(503, gin.H{"error": "history unavailable"})
			return
		}
		page, e := history.GetSequencePage(identity.FromContext(c.Request.Context()), uint(chatRoomID), cursor)
		if e != nil {
			c.JSON(serviceStatus(e, 500), gin.H{"error": e.Error()})
			return
		}
		c.JSON(200, page)
		return
	}
	if c.Query("snapshot_seq") != "" {
		c.JSON(400, gin.H{"error": "snapshot requires sequence cursor"})
		return
	}
	if chatRoomID == 0 || chatRoomID > math.MaxUint32 {
		c.JSON(400, gin.H{"error": "invalid room"})
		return
	}

	limitParam := c.Query("limit")
	beforeParam := c.Query("before_id")

	if limitParam == "" && beforeParam == "" {
		messages, err := mc.MessageService.GetMessagesByChatRoom(identity.FromContext(c.Request.Context()), uint(chatRoomID))
		if err != nil {
			c.JSON(serviceStatus(err, http.StatusInternalServerError), gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, messages)
		return
	}

	limit := 30
	if limitParam != "" {
		parsedLimit, err := strconv.Atoi(limitParam)
		if err != nil || parsedLimit <= 0 || parsedLimit > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		limit = parsedLimit
	}

	var beforeID uint64
	if beforeParam != "" {
		parsedBefore, err := strconv.ParseUint(beforeParam, 10, 63)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid before_id"})
			return
		}
		beforeID = parsedBefore
	}

	messages, err := mc.MessageService.GetMessagesPage(identity.FromContext(c.Request.Context()), uint(chatRoomID), uint(beforeID), limit)
	if err != nil {
		c.JSON(serviceStatus(err, http.StatusInternalServerError), gin.H{"error": err.Error()})
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

	if err := mc.MessageService.DeleteMessage(identity.FromContext(c.Request.Context()), uint(msgID)); err != nil {
		c.JSON(serviceStatus(err, http.StatusNotFound), gin.H{"error": err.Error()})
		return
	}

	c.AbortWithStatus(http.StatusNoContent)
}

// Consume the bounded body completely so trailing whitespace/extra JSON cannot bypass admission.
func jsonBindMessage(c *gin.Context, v any) error {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	return binding.JSON.BindBody(raw, v)
}
