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

/*
	chatrooms.POST("", chatRoomController.CreateChatRoom)
	chatrooms.GET("", chatRoomController.GetAllChatRooms)
	chatrooms.GET("/:id", chatRoomController.GetChatRoomByID)
	chatrooms.DELETE("/:id", chatRoomController.DeleteChatRoom)
	chatrooms.GET("/search", chatRoomController.SearchChatRooms)
*/
func TestChatroomCreate(t *testing.T) {
	r := setupBasicMiddleware(t)

	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockChatRoomService)
	chatRoomController := controller.NewChatRoomController(mockService)

	chatrooms := r.Group("/api/chatrooms")
	chatrooms.Use(loadsheddingFunc)
	chatrooms.Use(setupAuthMiddleware(t))
	{
		chatrooms.POST("", chatRoomController.CreateChatRoom)
	}

	//fail
	mockService.
		On("CreateChatRoom", "general").
		Return(&model.ChatRoom{}, errors.New("DB error")).
		Once()
	body := map[string]string{"name": "general"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/chatrooms", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	//bad request
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "error")

	//success
	chatRoom := &model.ChatRoom{ID: 1, Name: "general"}
	mockService.
		On("CreateChatRoom", "general").
		Return(chatRoom, nil).
		Once()
	body = map[string]string{"name": "general"}
	jsonBody, _ = json.Marshal(body)

	req = httptest.NewRequest(http.MethodPost, "/api/chatrooms", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), "general")
}

func TestChatroomGetAll(t *testing.T) {
	r := setupBasicMiddleware(t)

	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockChatRoomService)
	chatRoomController := controller.NewChatRoomController(mockService)

	chatrooms := r.Group("/api/chatrooms")
	chatrooms.Use(loadsheddingFunc)
	chatrooms.Use(setupAuthMiddleware(t))
	{
		chatrooms.GET("", chatRoomController.GetAllChatRooms)
	}

	//fail
	mockService.
		On("GetAllChatRooms").
		Return([]model.ChatRoom{}, errors.New("DB error")).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/api/chatrooms", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	//bad request
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "error")

	//success
	chatRoom := []model.ChatRoom{model.ChatRoom{ID: 1, Name: "general"}}
	mockService.
		On("GetAllChatRooms").
		Return(chatRoom, nil).
		Once()

	req = httptest.NewRequest(http.MethodGet, "/api/chatrooms", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "general")
}

func TestChatroomGetByID(t *testing.T) {
	r := setupBasicMiddleware(t)

	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockChatRoomService)
	chatRoomController := controller.NewChatRoomController(mockService)

	chatrooms := r.Group("/api/chatrooms")
	chatrooms.Use(loadsheddingFunc)
	chatrooms.Use(setupAuthMiddleware(t))
	{
		chatrooms.GET("/:id", chatRoomController.GetChatRoomByID)
	}

	//fail
	req := httptest.NewRequest(http.MethodGet, "/api/chatrooms/abc", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	//bad request
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "error")

	//success
	//chatRoom := new(model.ChatRoom{ID: 1, Name: "general"})
	mockService.
		On("GetChatRoomByID", uint(1)).
		Return(&model.ChatRoom{}, nil).
		Once()
	req = httptest.NewRequest(http.MethodGet, "/api/chatrooms/1", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "data")
}

func TestChatroomDelete(t *testing.T) {
	r := setupBasicMiddleware(t)

	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockChatRoomService)
	chatRoomController := controller.NewChatRoomController(mockService)

	chatrooms := r.Group("/api/chatrooms")
	chatrooms.Use(loadsheddingFunc)
	chatrooms.Use(setupAuthMiddleware(t))
	{
		chatrooms.DELETE("/:id", chatRoomController.DeleteChatRoom)
	}

	//fail
	req := httptest.NewRequest(http.MethodDelete, "/api/chatrooms/abc", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	//bad request
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "error")

	//success
	//chatRoom := new(model.ChatRoom{ID: 1, Name: "general"})
	mockService.
		On("DeleteChatRoom", uint(1)).
		Return(nil).
		Once()
	req = httptest.NewRequest(http.MethodDelete, "/api/chatrooms/1", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "deleted")
}

func TestChatroomSearch(t *testing.T) {
	r := setupBasicMiddleware(t)

	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockChatRoomService)
	chatRoomController := controller.NewChatRoomController(mockService)

	chatrooms := r.Group("/api/chatrooms")
	chatrooms.Use(loadsheddingFunc)
	chatrooms.Use(setupAuthMiddleware(t))
	{
		chatrooms.GET("/search", chatRoomController.SearchChatRooms)
	}

	//fail
	mockService.
		On("SearchChatRoomsByName", "cccchat").
		Return([]model.ChatRoom{}, errors.New("DB error")).
		Once()
	req := httptest.NewRequest(http.MethodGet, "/api/chatrooms/search?q=cccchat", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	//bad request
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "error")

	//success
	chatRoom := []model.ChatRoom{model.ChatRoom{ID: 1, Name: "general"}}
	mockService.
		On("SearchChatRoomsByName", "gen").
		Return(chatRoom, nil).
		Once()
	req = httptest.NewRequest(http.MethodGet, "/api/chatrooms/search?q=gen", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "general")
}

type MockChatRoomService struct {
	mock.Mock
}

func (m *MockChatRoomService) CreateChatRoom(name string) (*model.ChatRoom, error) {
	args := m.Called(name)
	return args.Get(0).(*model.ChatRoom), args.Error(1)
}

func (m *MockChatRoomService) GetAllChatRooms() ([]model.ChatRoom, error) {
	args := m.Called()
	return args.Get(0).([]model.ChatRoom), args.Error(1)
}

func (m *MockChatRoomService) GetChatRoomByID(id uint) (*model.ChatRoom, error) {
	args := m.Called(id)
	return args.Get(0).(*model.ChatRoom), args.Error(1)
}

func (m *MockChatRoomService) DeleteChatRoom(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockChatRoomService) GetChatRoomByName(name string) (*model.ChatRoom, error) {
	args := m.Called(name)
	return args.Get(0).(*model.ChatRoom), args.Error(1)
}

func (m *MockChatRoomService) SearchChatRoomsByName(keyword string) ([]model.ChatRoom, error) {
	args := m.Called(keyword)
	return args.Get(0).([]model.ChatRoom), args.Error(1)
}
