package controller_service_test

import(
    "testing"
    "time"
	"encoding/json"
//	"errors"
	"bytes"

//	"github.com/gin-contrib/cors"
//    "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
//	"github.com/stretchr/testify/mock"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
//    "gorm.io/driver/sqlite"
//    "gorm.io/gorm"

    "backend/internal/model"
    "backend/internal/repo"
    "backend/internal/service"
	"backend/internal/controller"
//	"backend/internal/cache"
//	"backend/internal/middleware/jwtauth"
	"backend/internal/middleware/loadshedding"
//	"backend/internal/middleware/logger"
//	"backend/internal/logrus"
//	"backend/utils"
)

func setupUserRoute(r *gin.Engine){
	db := setupTestDB()
	repos := repo.NewRepoContainer(db)
	userService := service.NewUserService(repos)
    userController := controller.NewUserController(userService)
	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)
	users := r.Group("/api/users")
	users.Use(loadsheddingFunc)
    // Apply middleware to all /chatrooms routes
	{
		// User routes
		users.POST("", userController.CreateUser)
		users.GET("", userController.GetUsers)
	}
}

func TestUserCRUD(t *testing.T){
	r := setupBasicMiddleware()
	setupUserRoute(r)

	body := model.User{Username:"BlackFrancis", Email:"pixies@gmail.com", Password:"test123"}
	jsonBody, _ := json.Marshal(body)

	//register new account
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Authorization", "Bearer good-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

    require.Equal(t, http.StatusCreated, w.Code)

	//test login
	req = httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
    require.Contains(t, w.Body.String(), `pixies`)
}