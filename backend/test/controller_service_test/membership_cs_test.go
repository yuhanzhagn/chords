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
	"backend/internal/cache"
//	"backend/internal/middleware/jwtauth"
	"backend/internal/middleware/loadshedding"
//	"backend/internal/middleware/logger"
//	"backend/internal/logrus"
//	"backend/utils"
)

func setupMembershipRoute(r *gin.Engine){
	db := setupTestDB()

	user := model.User{
		Username:  "testuser",
		Email:     "testuser@example.com",
		Password:  "securepassword",
		CreatedAt: time.Now(),
	}
	if err := db.Create(&user).Error; err != nil {
		return
	}
	chatRoom := model.ChatRoom{
		Name:      "Test Chat Room",
		CreatedAt: time.Now(),
	}
	if err := db.Create(&chatRoom).Error; err != nil {
		return
	}

	repos := repo.NewRepoContainer(db)
	typedCache := cache.NewTypedCache[[]model.ChatRoom](10*time.Minute, 15*time.Minute)
	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	s := service.NewMembershipService(repos, typedCache)

    membershipController := controller.NewMembershipController(s)
	memberships := r.Group("/api/memberships")
	memberships.Use(loadsheddingFunc)
    // Apply middleware to all /chatrooms routes
	{
        memberships.POST("/add-user", membershipController.AddUser)
		//memberships.POST("/chatrooms", membershipController.GetUserChatRooms)
		memberships.GET("/:username/chatrooms", func(c *gin.Context) {
   	 		username := c.Param("username")  // <-- string, no conversion
    		membershipController.GetUserChatRooms(c, username)
		})
    }
}

func TestMembershipCRUD(t *testing.T){
	r := setupBasicMiddleware()
	setupMembershipRoute(r)

	reqBody := map[string]interface{}{
		"username":   "testuser",
		"chatroomid": 1,
	}
	jsonBody, _ := json.Marshal(reqBody)
	//post new chatroom
	req := httptest.NewRequest(http.MethodPost, "/api/memberships/add-user", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Authorization", "Bearer good-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

    require.Equal(t, http.StatusOK, w.Code)

	//get all
	req = httptest.NewRequest(http.MethodGet, "/api/memberships/testuser/chatrooms", nil)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
    require.Contains(t, w.Body.String(), `Test`)

}