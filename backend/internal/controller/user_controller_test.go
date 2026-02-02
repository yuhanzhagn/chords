package controller_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"backend/internal/controller"
	"backend/internal/model"
)

func TestUserController_CreateUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockUserService)
	controller := &controller.UserController{Service: mockService}

	user := model.User{Username: "john", Email: "john@example.com", Password: "secret"}
	body, _ := json.Marshal(user)

	mockService.On("CreateUser", mock.AnythingOfType("*model.User")).Return(nil).Once()

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.CreateUser(ctx)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), "john")
	mockService.AssertExpectations(t)
}

func TestUserController_CreateUser_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	controller := &controller.UserController{Service: new(MockUserService)}

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer([]byte(`invalid`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.CreateUser(ctx)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "Invalid JSON")
}

func TestUserController_CreateUser_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockUserService)
	controller := &controller.UserController{Service: mockService}

	user := model.User{Username: "john", Email: "john@example.com", Password: "secret"}
	body, _ := json.Marshal(user)

	mockService.On("CreateUser", mock.AnythingOfType("*model.User")).Return(errors.New("db error")).Once()

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.CreateUser(ctx)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "Failed to create user")
	mockService.AssertExpectations(t)
}

func TestUserController_GetUsers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockUserService)
	controller := &controller.UserController{Service: mockService}

	users := []model.User{
		{ID: 1, Username: "john", Email: "john@example.com", CreatedAt: time.Now()},
		{ID: 2, Username: "jane", Email: "jane@example.com", CreatedAt: time.Now()},
	}

	mockService.On("GetAllUsers").Return(users, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.GetUsers(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "john")
	require.Contains(t, w.Body.String(), "jane")
	mockService.AssertExpectations(t)
}

func TestUserController_GetUsers_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockUserService)
	controller := &controller.UserController{Service: mockService}

	mockService.On("GetAllUsers").Return([]model.User{}, errors.New("db error")).Once()

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.GetUsers(ctx)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "Failed to fetch users")
	mockService.AssertExpectations(t)
}

// Mock UserService
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserService) GetAllUsers() ([]model.User, error) {
	args := m.Called()
	return args.Get(0).([]model.User), args.Error(1)
}

func (m *MockUserService) GetUserByUsername(username string) (*model.User, error) {
	args := m.Called(username)
	return args.Get(0).(*model.User), args.Error(1)
}
