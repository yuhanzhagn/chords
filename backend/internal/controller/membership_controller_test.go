package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"backend/internal/model"
)

func TestMembershipController_AddUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMembershipService)
	controller := NewMembershipController(mockService)

	reqBody := UserChatroom{
		Username:   "john",
		ChatRoomID: 1,
	}

	mockService.
		On("AddUserToChatRoom", "john", uint(1)).
		Return(nil).
		Once()

	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/membership", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.AddUser(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "user added to chatroom")
	require.Contains(t, w.Body.String(), `"john"`)

	mockService.AssertExpectations(t)
}

func TestMembershipController_AddUser_InvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	controller := NewMembershipController(new(MockMembershipService))

	req := httptest.NewRequest(http.MethodPost, "/membership", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.AddUser(ctx)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMembershipController_AddUser_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMembershipService)
	controller := NewMembershipController(mockService)

	reqBody := UserChatroom{
		Username:   "john",
		ChatRoomID: 99,
	}

	mockService.
		On("AddUserToChatRoom", "john", uint(99)).
		Return(errors.New("chatroom not found")).
		Once()

	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/membership", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.AddUser(ctx)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "chatroom not found")

	mockService.AssertExpectations(t)
}

func TestMembershipController_GetUserChatRooms_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMembershipService)
	controller := NewMembershipController(mockService)

	rooms := []model.ChatRoom{
		{ID: 1, Name: "general"},
		{ID: 2, Name: "random"},
	}

	mockService.
		On("GetUserSubscribedChatRooms", "john").
		Return(rooms, nil).
		Once()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	controller.GetUserChatRooms(ctx, "john")

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "general")
	require.Contains(t, w.Body.String(), "random")

	mockService.AssertExpectations(t)
}

func TestMembershipController_GetUserChatRooms_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMembershipService)
	controller := NewMembershipController(mockService)

	mockService.
		On("GetUserSubscribedChatRooms", "john").
		Return([]model.ChatRoom{}, errors.New("db error")).
		Once()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	controller.GetUserChatRooms(ctx, "john")

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "db error")

	mockService.AssertExpectations(t)
}

type MockMembershipService struct {
	mock.Mock
}

func (m *MockMembershipService) AddUserToChatRoom(username string, chatRoomID uint) error {
	args := m.Called(username, chatRoomID)
	return args.Error(0)
}

func (m *MockMembershipService) GetUserSubscribedChatRooms(username string) ([]model.ChatRoom, error) {
	args := m.Called(username)
	return args.Get(0).([]model.ChatRoom), args.Error(1)
}

func (m *MockMembershipService) GetUserChatRoomsFromDB(username string) ([]model.ChatRoom, error) {
	args := m.Called(username)
	return args.Get(0).([]model.ChatRoom), args.Error(1)
}
