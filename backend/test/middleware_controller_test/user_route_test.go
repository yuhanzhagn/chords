package middleware_controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	//	"github.com/gin-contrib/cors"
	//	"github.com/stretchr/testify/assert"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	//   "gorm.io/driver/sqlite"
	//    "gorm.io/gorm"

	"backend/internal/model"
	//    "backend/internal/service"
	"backend/internal/controller"
	//	"backend/internal/cache"
	"backend/internal/middleware/jwtauth"
	"backend/internal/middleware/loadshedding"
	//	"backend/internal/middleware/logger"
	//	"backend/internal/logrus"
	//	"backend/utils"
)

func setupAuthMiddleware(t *testing.T) gin.HandlerFunc {
	mockService := new(MockAuthService)
	mockService.
		On("ValidateJWT", "correct").
		Return(new(model.User), new(model.UserSession), nil).
		Twice()
	return jwtauth.NewAuthMiddleware(mockService).Auth()
}

func TestUserCreate(t *testing.T) {
	r := setupBasicMiddleware(t)

	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockUserService)
	userController := &controller.UserController{Service: mockService}

	userRoute := r.Group("/api/users")
	userRoute.Use(loadsheddingFunc)
	userRoute.Use(setupAuthMiddleware(t))
	{
		userRoute.POST("", userController.CreateUser)
		userRoute.GET("", userController.GetUsers)
	}

	//fail
	user := model.User{Username: "john", Password: "secret"}
	jsonBody, _ := json.Marshal(user)

	mockService.On("CreateUser", mock.AnythingOfType("*model.User")).Once().Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "error")

	//success
	user = model.User{Username: "john", Email: "john@example.com", Password: "secret"}
	jsonBody, _ = json.Marshal(user)
	//mockService.On("CreateUser", mock.AnythingOfType("*model.User")).Return(nil).Once()

	req = httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), "john")
	mockService.AssertExpectations(t)
}

func TestUserGet(t *testing.T) {
	r := setupBasicMiddleware(t)

	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockUserService)
	userController := &controller.UserController{Service: mockService}

	userRoute := r.Group("/api/users")
	userRoute.Use(loadsheddingFunc)
	userRoute.Use(setupAuthMiddleware(t))
	{
		userRoute.POST("", userController.CreateUser)
		userRoute.GET("", userController.GetUsers)
	}

	//fail due to db error
	mockService.On("GetAllUsers").Return([]model.User{}, errors.New("DB error")).Once()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "error")

	//success
	user := model.User{Username: "john", Email: "john@example.com", Password: "secret"}
	mockService.On("GetAllUsers").Return([]model.User{user}, nil).Once()

	req = httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "john")
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
