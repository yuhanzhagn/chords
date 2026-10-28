package controller_service_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/controller"
	"backend/internal/middleware/jwtauth"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
	"backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountSecurity(t *testing.T) {
	db := setupTestDB()
	repos := repo.NewRepoContainer(db)
	auth := service.NewAuthService(repos, nil)
	ac := controller.NewAuthController(auth)
	uc := controller.NewUserController(service.NewUserService(repos))
	r := gin.New()
	r.POST("/register", ac.Register)
	r.POST("/login", ac.Login)
	users := r.Group("/users", jwtauth.NewAuthMiddleware(auth).Auth())
	users.POST("", uc.CreateUser)
	users.GET("", uc.GetUsers)
	request := func(method, path, body, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	require.Equal(t, 200, request("POST", "/register", `{"username":"alice","email":"alice@test.com","password":"secret"}`, "").Code)
	login := request("POST", "/login", `{"username":"alice","password":"secret"}`, "")
	require.Equal(t, 200, login.Code)
	var credentials struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(login.Body.Bytes(), &credentials))
	oldToken := credentials.Token
	for _, body := range []string{`{"username":"alice","password":"wrong"}`, `{"username":"unknown","password":"secret"}`, `{"username":"alice"}`, `{`} {
		require.GreaterOrEqual(t, request("POST", "/login", body, "").Code, 400)
		require.Equal(t, 200, request("GET", "/users", "", oldToken).Code, body)
	}
	created := request("POST", "/users", `{"id":999,"username":"bob","email":"bob@test.com","password":"secret","sessions":[{"SessionID":"injected","RefreshToken":"secret-refresh"}]}`, oldToken)
	require.Equal(t, 201, created.Code, created.Body.String())
	for _, response := range []*httptest.ResponseRecorder{created, request("GET", "/users", "", oldToken)} {
		for _, forbidden := range []string{"password", "sessions", "RefreshToken", "secret-refresh", "$2a$"} {
			require.NotContains(t, response.Body.String(), forbidden)
		}
	}
	for _, name := range []string{"alice", "bob"} {
		user, err := repos.User.GetByUsername(name)
		require.NoError(t, err)
		require.NotEqual(t, uint(999), user.ID)
		require.True(t, utils.CheckPasswordHash("secret", user.Password), name)
	}
	var injected int64
	require.NoError(t, db.Model(&model.UserSession{}).Where("session_id = ?", "injected").Count(&injected).Error)
	require.Zero(t, injected)
	for _, path := range []string{"/register", "/users"} {
		body := `{"username":"too-long","email":"long@test.com","password":"` + strings.Repeat("a", 73) + `"}`
		require.Equal(t, 400, request("POST", path, body, oldToken).Code)
		var count int64
		require.NoError(t, db.Model(&model.User{}).Where("username = ?", "too-long").Count(&count).Error)
		require.Zero(t, count)
	}
	require.Equal(t, 200, request("POST", "/login", `{"username":"alice","password":"secret"}`, "").Code)
	require.Equal(t, 401, request("GET", "/users", "", oldToken).Code)
}
