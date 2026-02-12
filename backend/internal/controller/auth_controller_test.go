package controller

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

	"backend/internal/model"
)

func TestAuthController_Register_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	controller := NewAuthController(mockService)

	body := map[string]string{
		"username": "john",
		"email":    "john@test.com",
		"password": "secret",
	}
	jsonBody, _ := json.Marshal(body)

	mockService.
		On("Register", mock.AnythingOfType("*model.User")).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.Register(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Registered")
	mockService.AssertExpectations(t)
}

func TestAuthController_Register_InvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	controller := NewAuthController(mockService)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer([]byte("invalid")))
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.Register(ctx)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "Invalid input")
}

func TestAuthController_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	controller := NewAuthController(mockService)

	body := map[string]string{
		"username": "john",
		"password": "secret",
	}
	jsonBody, _ := json.Marshal(body)

	mockService.
		On("GetUserIDByUsername", "john").
		Return(uint(1), nil)

	mockService.
		On("ForceLogoutAll", uint(1)).
		Return(nil)

	mockService.
		On("Login", "john", "secret").
		Return(uint(1), "token", "jti", "sessionID", nil)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.Login(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"token"`)
	require.Contains(t, w.Body.String(), `"sessionID"`)

	mockService.AssertExpectations(t)
}

func TestAuthController_Login_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	controller := NewAuthController(mockService)

	body := map[string]string{
		"username": "john",
		"password": "wrong",
	}
	jsonBody, _ := json.Marshal(body)

	mockService.
		On("GetUserIDByUsername", "john").
		Return(uint(1), nil)

	mockService.
		On("ForceLogoutAll", uint(1)).
		Return(nil)

	mockService.
		On("Login", "john", "wrong").
		Return(uint(0), "", "", "", errors.New("invalid credentials"))

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.Login(ctx)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthController_Logout_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	controller := NewAuthController(mockService)

	mockService.
		On("Logout", "token123").
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.Header.Set("Authorization", "Bearer token123")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.Logout(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "logout successful")
	mockService.AssertExpectations(t)
}

func TestAuthController_Logout_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	controller := NewAuthController(new(MockAuthService))

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	controller.Logout(ctx)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "missing Authorization")
}

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockAuthService) Login(username, password string) (uint, string, string, string, error) {
	args := m.Called(username, password)
	return args.Get(0).(uint),
		args.Get(1).(string),
		args.Get(2).(string),
		args.Get(3).(string),
		args.Error(4)
}

func (m *MockAuthService) Logout(tokenStr string) error {
	args := m.Called(tokenStr)
	return args.Error(0)
}

func (m *MockAuthService) ValidateJWT(tokenStr string) (*model.User, *model.UserSession, error) {
	args := m.Called(tokenStr)
	return args.Get(0).(*model.User),
		args.Get(1).(*model.UserSession),
		args.Error(2)
}

func (m *MockAuthService) ForceLogoutAll(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockAuthService) ForceLogoutOne(userID uint, sessionID string) error {
	args := m.Called(userID, sessionID)
	return args.Error(0)
}

func (m *MockAuthService) GetUserIDByUsername(username string) (uint, error) {
	args := m.Called(username)
	return args.Get(0).(uint), args.Error(1)
}

func (m *MockAuthService) Block(jti string, exp time.Time) error {
	args := m.Called(jti, exp)
	return args.Error(0)
}

func (m *MockAuthService) IsBlocked(jti string) bool {
	args := m.Called(jti)
	return args.Bool(0)
}

func (m *MockAuthService) Clean(jti string) {
	m.Called(jti)
}
