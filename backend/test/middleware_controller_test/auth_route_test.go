package middleware_controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"backend/internal/cache"
	"backend/internal/controller"
	"backend/internal/model"
	"backend/internal/service"

	//	"backend/internal/middleware/jwtauth"
	"backend/internal/logrus"
	"backend/internal/middleware/loadshedding"
	"backend/internal/middleware/logger"
	//	"backend/utils"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err, "failed to connect database")

	// Migrate schema
	err = db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.Message{}, &model.ChatRoom{}, &model.UserChatRoom{})
	assert.NoError(t, err, "failed to migrate database")

	return db
}

func setupTestCache(t *testing.T) *cache.TypedCache[service.BlockEntry] {
	c := cache.NewTypedCache[service.BlockEntry](
		10*time.Minute,
		15*time.Minute,
	)
	return c
}

func setupBasicMiddleware(t *testing.T) *gin.Engine {
	r := gin.New()
	// init middlewares
	log := logrus.InitLogrus()
	r.Use(logger.LogrusLogger(log))
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // or "*" for all origins
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.Group("/api")
	return r
}

func TestAuthLogin(t *testing.T) {
	r := setupBasicMiddleware(t)

	// authFunc := jwtauth.NewAuthMiddleware(authService).Auth()
	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockAuthService)
	authController := controller.NewAuthController(mockService)

	authRoutes := r.Group("/api/auth")
	authRoutes.Use(loadsheddingFunc)
	{
		authRoutes.POST("/register", authController.Register)
		authRoutes.POST("/login", authController.Login)
		authRoutes.POST("/logout", authController.Logout)
	}

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

	mockService.
		On("Login", "john", "correct").
		Return(uint(1), "token", "jti", "sessionid", nil)
	//fail
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Authorization", "Bearer good-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), `"error"`)

	//success
	body = map[string]string{
		"username": "john",
		"password": "correct",
	}

	jsonBody, _ = json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"token"`)
}

func TestAuthRegister(t *testing.T) {
	r := setupBasicMiddleware(t)

	// authFunc := jwtauth.NewAuthMiddleware(authService).Auth()
	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockAuthService)
	authController := controller.NewAuthController(mockService)

	authRoutes := r.Group("/api/auth")
	authRoutes.Use(loadsheddingFunc)
	{
		authRoutes.POST("/register", authController.Register)
		//authRoutes.POST("/login", authController.Login)
		//authRoutes.POST("/logout", authController.Logout)
	}

	body := map[string]string{
		"username": "john",
		"gmail":    "crazyjohn@test.com",
		"password": "wrong",
	}
	jsonBody, _ := json.Marshal(body)

	mockService.
		On("Register", mock.Anything).
		Return(nil)

	//fail
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Authorization", "Bearer good-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	//require.Contains(t, w.Body.String(), `"error"`)

	//success
	body = map[string]string{
		"username": "john",
		"email":    "crazyjohn@test.com",
		"password": "correct",
	}

	jsonBody, _ = json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	//require.Contains(t, w.Body.String(), `"token"`)
}

func TestAuthLogout(t *testing.T) {
	r := setupBasicMiddleware(t)

	// authFunc := jwtauth.NewAuthMiddleware(authService).Auth()
	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	mockService := new(MockAuthService)
	authController := controller.NewAuthController(mockService)

	authRoutes := r.Group("/api/auth")
	authRoutes.Use(loadsheddingFunc)
	{
		//authRoutes.POST("/register", authController.Register)
		//authRoutes.POST("/login", authController.Login)
		authRoutes.POST("/logout", authController.Logout)
	}
	mockService.
		On("Logout", "Wrong").
		Return(errors.New("invalid credentials"))
	mockService.
		On("Logout", "Correct").
		Return(nil)

	//fail
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", bytes.NewBuffer(nil))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer Wrong")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), `"error"`)

	//success
	req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", bytes.NewBuffer(nil))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer Correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	//require.Contains(t, w.Body.String(), `"token"`)
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
