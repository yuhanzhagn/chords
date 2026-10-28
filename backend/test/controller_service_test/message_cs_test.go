package controller_service_test

import (
	"encoding/json"
	"testing"
	"time"
	//	"errors"
	"bytes"

	//	"github.com/gin-contrib/cors"
	//    "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	//	"github.com/stretchr/testify/mock"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	//    "gorm.io/driver/sqlite"
	//    "gorm.io/gorm"

	"backend/internal/controller"
	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
	//	"backend/internal/cache"
	//	"backend/internal/middleware/jwtauth"
	"backend/internal/middleware/loadshedding"
	//	"backend/internal/middleware/logger"
	//	"backend/internal/logrus"
	//	"backend/utils"
)

func setupMessageRoute(r *gin.Engine) {
	db := setupTestDB()
	repos := repo.NewRepoContainer(db)
	if err := db.Create(&model.ChatRoom{ID: 1, Name: "General"}).Error; err != nil {
		panic(err)
	}
	if err := db.Create(&model.UserChatRoom{UserID: 1, ChatRoomID: 1}).Error; err != nil {
		panic(err)
	}
	if err := db.Create(&model.UserSession{UserID: 1, SessionID: "session", ExpiresAt: time.Now().Add(time.Hour)}).Error; err != nil {
		panic(err)
	}
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(identity.WithContext(c.Request.Context(), identity.New(1, "alice", "session")))
		c.Next()
	})
	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	messageService := service.NewMessageService(repos)
	messageController := controller.NewMessageController(messageService)

	//auth := jwtauth.JWTAuthMiddleware()

	r.POST("/api/messages", loadsheddingFunc, messageController.CreateMessage)
	r.GET("/api/chatrooms/:id/messages", loadsheddingFunc, messageController.GetMessagesByChatRoom)
	r.DELETE("/api/messages/:id", loadsheddingFunc, messageController.DeleteMessage)
}

func TestMessageCRUD(t *testing.T) {
	r := setupBasicMiddleware()
	setupMessageRoute(r)

	reqBody := map[string]interface{}{
		"content":      "Where is my mind",
		"user_id":      1,
		"chat_room_id": 1,
	}
	jsonBody, _ := json.Marshal(reqBody)
	//post new chatroom
	req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Authorization", "Bearer good-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	//get all
	req = httptest.NewRequest(http.MethodGet, "/api/chatrooms/1/messages", nil)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `mind`)

	//delete
	req = httptest.NewRequest(http.MethodDelete, "/api/messages/1", nil)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

}
