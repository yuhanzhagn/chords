package controller

import (
	"bytes"
	"encoding/json"
//	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"backend/internal/model"
)

func TestChatRoomController_CreateChatRoom_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockChatRoomService)
	controller := NewChatRoomController(mockService)

	chatRoom := &model.ChatRoom{ID: 1, Name: "general"}

	mockService.
		On("CreateChatRoom", "general").
		Return(chatRoom, nil).
		Once()

	body := map[string]string{"name": "general"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/chatrooms", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.CreateChatRoom(ctx)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), `"general"`)
	mockService.AssertExpectations(t)
}


func TestChatRoomController_CreateChatRoom_InvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	controller := NewChatRoomController(new(MockChatRoomService))

	req := httptest.NewRequest(http.MethodPost, "/chatrooms", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.CreateChatRoom(ctx)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChatRoomController_GetAllChatRooms_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockChatRoomService)
	controller := NewChatRoomController(mockService)

	rooms := []model.ChatRoom{
		{ID: 1, Name: "general"},
		{ID: 2, Name: "random"},
	}

	mockService.
		On("GetAllChatRooms").
		Return(rooms, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/chatrooms", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.GetAllChatRooms(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "general")
	mockService.AssertExpectations(t)
}

func TestChatRoomController_GetChatRoomByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockChatRoomService)
	controller := NewChatRoomController(mockService)

	room := &model.ChatRoom{ID: 1, Name: "general"}

	mockService.
		On("GetChatRoomByID", uint(1)).
		Return(room, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/chatrooms/1", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	ctx.Request = req

	controller.GetChatRoomByID(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "general")
	mockService.AssertExpectations(t)
}

func TestChatRoomController_GetChatRoomByID_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	controller := NewChatRoomController(new(MockChatRoomService))

	req := httptest.NewRequest(http.MethodGet, "/chatrooms/abc", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: "abc"}}
	ctx.Request = req

	controller.GetChatRoomByID(ctx)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChatRoomController_DeleteChatRoom_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockChatRoomService)
	controller := NewChatRoomController(mockService)

	mockService.
		On("DeleteChatRoom", uint(1)).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/chatrooms/1", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	ctx.Request = req

	controller.DeleteChatRoom(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "deleted")
	mockService.AssertExpectations(t)
}

func TestChatRoomController_SearchChatRooms_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockChatRoomService)
	controller := NewChatRoomController(mockService)

	rooms := []model.ChatRoom{
		{ID: 1, Name: "general"},
	}

	mockService.
		On("SearchChatRoomsByName", "gen").
		Return(rooms, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/chatrooms/search?q=gen", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.SearchChatRooms(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "general")
	mockService.AssertExpectations(t)
}

func TestChatRoomController_SearchChatRooms_MissingQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	controller := NewChatRoomController(new(MockChatRoomService))

	req := httptest.NewRequest(http.MethodGet, "/chatrooms/search", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.SearchChatRooms(ctx)

	require.Equal(t, http.StatusBadRequest, w.Code)
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