package middleware_controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	//	"github.com/gin-contrib/cors"
	//	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	//	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	//   "gorm.io/driver/sqlite"
	//    "gorm.io/gorm"

	"backend/internal/model"
	//    "backend/internal/service"
	"backend/internal/controller"
	//	"backend/internal/cache"
	//	"backend/internal/middleware/jwtauth"
	"backend/internal/middleware/loadshedding"
	//	"backend/internal/middleware/logger"
	//	"backend/internal/logrus"
	//	"backend/utils"
)

type Msg struct {
	Content    string `json:"content"`
	UserID     uint   `json:"user_id"`
	ChatRoomID uint   `json:"chat_room_id"`
}

func TestMeesageCreate(t *testing.T) {
	r := setupBasicMiddleware(t)

	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockMessageService)
	messageController := &controller.MessageController{MessageService: mockService}

	messageRoute := r.Group("/api")
	messageRoute.Use(loadsheddingFunc)
	messageRoute.Use(setupAuthMiddleware(t))
	{
		messageRoute.POST("/messages", messageController.CreateMessage)
	}

	//fail
	msg := Msg{Content: "Schizophrenia is taking me home", UserID: uint(1), ChatRoomID: uint(1)}
	jsonBody, _ := json.Marshal(msg)

	mockService.On("CreateMessage", mock.AnythingOfType("*model.Message")).Once().Return(errors.New("DB error"))

	req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "error")

	//success
	// msg := model.User{Content: "Schizophrenia is taking me home", UserID: uint(1), ChatRoomID: uint(1)}
	// jsonBody, _ = json.Marshal(msg)
	mockService.On("CreateMessage", mock.AnythingOfType("*model.Message")).Once().Return(nil)

	req = httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), "Schizophrenia")
	//mockService.AssertExpectations(t)
}

func TestGetMessageByChatRoom(t *testing.T) {
	r := setupBasicMiddleware(t)

	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockMessageService)
	messageController := &controller.MessageController{MessageService: mockService}

	messageRoute := r.Group("/api")
	messageRoute.Use(loadsheddingFunc)
	messageRoute.Use(setupAuthMiddleware(t))
	{
		messageRoute.GET("/chatrooms/:id/messages", messageController.GetMessagesByChatRoom)
	}

	//fail
	mockService.
		On("GetMessagesByChatRoom", uint(1)).
		Once().
		Return([]model.Message{}, errors.New("DB error"))

	req := httptest.NewRequest(http.MethodGet, "/api/chatrooms/1/messages", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "error")

	//success
	// msg := model.User{Content: "Schizophrenia is taking me home", UserID: uint(1), ChatRoomID: uint(1)}
	// jsonBody, _ = json.Marshal(msg)
	mockService.
		On("GetMessagesByChatRoom", uint(1)).
		Once().
		Return([]model.Message{model.Message{Content: "Where is my mind"}}, nil)

	req = httptest.NewRequest(http.MethodGet, "/api/chatrooms/1/messages", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Where")
	//mockService.AssertExpectations(t)
}

func TestMessageDelete(t *testing.T) {
	r := setupBasicMiddleware(t)

	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockMessageService)
	messageController := &controller.MessageController{MessageService: mockService}

	messageRoute := r.Group("/api")
	messageRoute.Use(loadsheddingFunc)
	messageRoute.Use(setupAuthMiddleware(t))
	{
		messageRoute.DELETE("/messages/:id", messageController.DeleteMessage)
	}

	//fail
	mockService.
		On("DeleteMessage", uint(1)).
		Once().
		Return(errors.New("DB error"))

	req := httptest.NewRequest(http.MethodDelete, "/api/messages/1", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "error")

	//success
	// msg := model.User{Content: "Schizophrenia is taking me home", UserID: uint(1), ChatRoomID: uint(1)}
	// jsonBody, _ = json.Marshal(msg)
	mockService.
		On("DeleteMessage", uint(1)).
		Once().
		Return(nil)

	req = httptest.NewRequest(http.MethodDelete, "/api/messages/1", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	//mockService.AssertExpectations(t)
}

type MockMessageService struct {
	mock.Mock
}

func (m *MockMessageService) CreateMessage(msg *model.Message) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *MockMessageService) GetMessagesByChatRoom(chatRoomID uint) ([]model.Message, error) {
	args := m.Called(chatRoomID)
	return args.Get(0).([]model.Message), args.Error(1)
}

func (m *MockMessageService) DeleteMessage(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockMessageService) GetMessagesWithLimit(roomID uint, limit int) ([]model.Message, error) {
	args := m.Called(roomID, limit)
	return args.Get(0).([]model.Message), args.Error(1)
}
