package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
//	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"backend/internal/model"
)

func TestMessageController_CreateMessage_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMessageService)
	controller := NewMessageController(mockService)

	reqBody := map[string]interface{}{
		"content":     "Hello world",
		"user_id":     1,
		"chat_room_id": 2,
	}
	body, _ := json.Marshal(reqBody)

	mockService.
		On("CreateMessage", mock.AnythingOfType("*model.Message")).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.CreateMessage(ctx)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), "Hello world")
	mockService.AssertExpectations(t)
}

func TestMessageController_CreateMessage_InvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	controller := NewMessageController(new(MockMessageService))

	req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.CreateMessage(ctx)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMessageController_CreateMessage_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMessageService)
	controller := NewMessageController(mockService)

	reqBody := map[string]interface{}{
		"content":     "Hello world",
		"user_id":     1,
		"chat_room_id": 2,
	}
	body, _ := json.Marshal(reqBody)

	mockService.
		On("CreateMessage", mock.AnythingOfType("*model.Message")).
		Return(errors.New("db error")).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.CreateMessage(ctx)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "db error")
	mockService.AssertExpectations(t)
}

func TestMessageController_GetMessagesByChatRoom_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMessageService)
	controller := NewMessageController(mockService)

	messages := []model.Message{
		{ID: 1, Content: "Hello", UserID: 1, ChatRoomID: 2, CreatedAt: time.Now()},
		{ID: 2, Content: "World", UserID: 2, ChatRoomID: 2, CreatedAt: time.Now()},
	}

	mockService.
		On("GetMessagesByChatRoom", uint(2)).
		Return(messages, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/chatrooms/2/messages", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: "2"}}
	ctx.Request = req

	controller.GetMessagesByChatRoom(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Hello")
	require.Contains(t, w.Body.String(), "World")
	mockService.AssertExpectations(t)
}

func TestMessageController_GetMessagesByChatRoom_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	controller := NewMessageController(new(MockMessageService))

	req := httptest.NewRequest(http.MethodGet, "/chatrooms/abc/messages", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: "abc"}}
	ctx.Request = req

	controller.GetMessagesByChatRoom(ctx)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMessageController_DeleteMessage_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMessageService)
	controller := NewMessageController(mockService)

	mockService.On("DeleteMessage", uint(1)).Return(nil).Once()

	req := httptest.NewRequest(http.MethodDelete, "/messages/1", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	ctx.Request = req

	controller.DeleteMessage(ctx)

	require.Equal(t, http.StatusNoContent, w.Code)
	mockService.AssertExpectations(t)
}

func TestMessageController_DeleteMessage_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	controller := NewMessageController(new(MockMessageService))

	req := httptest.NewRequest(http.MethodDelete, "/messages/abc", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: "abc"}}
	ctx.Request = req

	controller.DeleteMessage(ctx)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMessageController_DeleteMessage_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMessageService)
	controller := NewMessageController(mockService)

	mockService.On("DeleteMessage", uint(99)).Return(errors.New("not found")).Once()

	req := httptest.NewRequest(http.MethodDelete, "/messages/99", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: "99"}}
	ctx.Request = req

	controller.DeleteMessage(ctx)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "not found")
	mockService.AssertExpectations(t)
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
