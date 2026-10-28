package controller_service_test

import (
	"backend/internal/identity"
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

func setupChatRoomRoute(t *testing.T, r *gin.Engine) {
	db := setupTestDB()
	repos := repo.NewRepoContainer(db)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "alice", Email: "a@test.com", Password: "hash"}).Error)
	require.NoError(t, db.Create(&model.UserSession{UserID: 1, SessionID: "session", ExpiresAt: time.Now().Add(time.Hour)}).Error)
	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	chatRoomService := service.NewChatRoomService(repos)
	chatRoomController := controller.NewChatRoomController(chatRoomService)

	chatrooms := r.Group("/api/chatrooms")

	//	load shed
	chatrooms.Use(loadsheddingFunc)
	chatrooms.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(identity.WithContext(c.Request.Context(), identity.New(1, "alice", "session")))
		c.Next()
	})
	// Apply middleware to all /chatrooms routes
	{
		chatrooms.POST("", chatRoomController.CreateChatRoom)
		chatrooms.GET("", chatRoomController.GetAllChatRooms)
		chatrooms.GET("/:id", chatRoomController.GetChatRoomByID)
		chatrooms.DELETE("/:id", chatRoomController.DeleteChatRoom)
		chatrooms.GET("/search", chatRoomController.SearchChatRooms)
	}

}

func TestChatRoomCRUD(t *testing.T) {
	r := setupBasicMiddleware()
	setupChatRoomRoute(t, r)

	body := model.ChatRoom{Name: "Noise"}
	jsonBody, _ := json.Marshal(body)

	//post new chatroom
	req := httptest.NewRequest(http.MethodPost, "/api/chatrooms", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Authorization", "Bearer good-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	//get all
	req = httptest.NewRequest(http.MethodGet, "/api/chatrooms", nil)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `Noise`)

	// get by id
	req = httptest.NewRequest(http.MethodGet, "/api/chatrooms/1", nil)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `se`)

	//search
	req = httptest.NewRequest(http.MethodGet, "/api/chatrooms/search?q=Noi", nil)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `ois`)

	//delete
	req = httptest.NewRequest(http.MethodDelete, "/api/chatrooms/1", nil)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
