package controller_service_test

import (
	"encoding/json"
	"testing"
	"time"

	//	"errors"
	"bytes"

	"github.com/gin-contrib/cors"
	//    "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	//	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"backend/internal/cache"
	"backend/internal/controller"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"

	//	"backend/internal/middleware/jwtauth"
	"backend/internal/logrus"
	"backend/internal/middleware/loadshedding"
	"backend/internal/middleware/logger"
	//	"backend/utils"
)

func setupTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	//assert.NoError(t, err, "failed to connect database")

	// Migrate schema
	_ = db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.Message{}, &model.ChatRoom{}, &model.UserChatRoom{})
	//assert.NoError(t, err, "failed to migrate database")

	return db
}

func setupBasicMiddleware() *gin.Engine {
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
	return r
}

func setupAuthRoute(r *gin.Engine) {
	db := setupTestDB()
	repos := repo.NewRepoContainer(db)
	typedCache := cache.NewTypedCache[service.BlockEntry](10*time.Minute, 15*time.Minute)
	authService := service.NewAuthService(repos, typedCache)
	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	authController := controller.NewAuthController(authService)

	authRoutes := r.Group("/api/auth")
	authRoutes.Use(loadsheddingFunc)
	{
		authRoutes.POST("/register", authController.Register)
		authRoutes.POST("/login", authController.Login)
		authRoutes.POST("/logout", authController.Logout)
	}
}

func TestRegister(t *testing.T) {
	r := setupBasicMiddleware()
	setupAuthRoute(r)

	body := map[string]string{
		"username": "john",
		"email":    "crazyjohn@test.com",
		"password": "correct",
	}
	jsonBody, _ := json.Marshal(body)

	//register new account
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Authorization", "Bearer good-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	//test login
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

	// test logout
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(w.Body.String()), &m); err != nil {
		require.NoError(t, err)
	}
	token := m["token"].(string)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", bytes.NewBuffer(nil))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
