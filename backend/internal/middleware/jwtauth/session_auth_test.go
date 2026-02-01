package jwtauth_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
//	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/mock"

//	"backend/internal/service"
	"backend/internal/model"
	"backend/internal/middleware/jwtauth"
)

func setupRouter(mw gin.HandlerFunc) *gin.Engine {
    gin.SetMode(gin.TestMode)

    r := gin.New()
    r.Use(mw)
    r.GET("/protected", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"ok": true})
    })
    return r
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
    authSvc := &MockAuthService{}
    mw := &jwtauth.AuthMiddleware{AuthSvc: authSvc}

    router := setupRouter(mw.Auth())

    req := httptest.NewRequest(http.MethodGet, "/protected", nil)
    w := httptest.NewRecorder()

    router.ServeHTTP(w, req)

    require.Equal(t, http.StatusUnauthorized, w.Code)
    require.Contains(t, w.Body.String(), "Missing or invalid Authorization header")
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
    authSvc := &MockAuthService{
        validateFn: func(token string) error {
            return errors.New("invalid token")
        },
    }
    mw := &jwtauth.AuthMiddleware{AuthSvc: authSvc}

	authSvc.On("ValidateJWT", "bad-token").Return(nil, nil, errors.New("invalid token"))

    router := setupRouter(mw.Auth())

    req := httptest.NewRequest(http.MethodGet, "/protected", nil)
    req.Header.Set("Authorization", "Bearer bad-token")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    require.Equal(t, http.StatusUnauthorized, w.Code)
    require.Contains(t, w.Body.String(), "invalid token")
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
    authSvc := &MockAuthService{
        validateFn: func(token string) error {
            require.Equal(t, "good-token", token)
            return nil
        },
    }
    mw := &jwtauth.AuthMiddleware{AuthSvc: authSvc}
	authSvc.On("ValidateJWT", "good-token").Return(nil, nil, nil)
    router := setupRouter(mw.Auth())

    req := httptest.NewRequest(http.MethodGet, "/protected", nil)
    req.Header.Set("Authorization", "Bearer good-token")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    require.Equal(t, http.StatusOK, w.Code)
    require.Contains(t, w.Body.String(), `"ok":true`)
}

type MockAuthService struct {
	mock.Mock
	validateFn func(token string) error
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

	var user *model.User
	if u := args.Get(0); u != nil {
		user = u.(*model.User)
	}

	var session *model.UserSession
	if s := args.Get(1); s != nil {
		session = s.(*model.UserSession)
	}

	return user, session, args.Error(2)
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