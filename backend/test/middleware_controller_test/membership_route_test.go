package middleware_controller

import(
    "testing"
    "time"
	"encoding/json"
	"errors"
	"bytes"

//	"github.com/gin-contrib/cors"
//	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/mock"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
//   "gorm.io/driver/sqlite"
//    "gorm.io/gorm"

    "backend/internal/model"
//    "backend/internal/service"
	"backend/internal/controller"
//	"backend/internal/cache"
//	"backend/internal/middleware/jwtauth"
	"backend/internal/middleware/loadshedding"
//	"backend/internal/middleware/logger"
//	"backend/internal/logrus"
//	"backend/utils"
)

func TestAddUserToChatRoom(t *testing.T) {
    r := setupBasicMiddleware(t)

	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)
 
	mockService := new(MockMembershipService)
	membershipController := controller.NewMembershipController(mockService)

	membershipRoute := r.Group("/api/memberships")
	membershipRoute.Use(loadsheddingFunc)
	membershipRoute.Use(setupAuthMiddleware(t))
    {
		membershipRoute.POST("/add-user", membershipController.AddUser)
    } 

	//fail
	msg := controller.UserChatroom{Username: "Thom Yorke", ChatRoomID: uint(1)}
	jsonBody, _ := json.Marshal(msg)

	mockService.
	On("AddUserToChatRoom", "Thom Yorke", uint(1)).
	Once().
	Return(errors.New("DB error"))

	req := httptest.NewRequest(http.MethodPost, "/api/memberships/add-user", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "error")

	//success
	// msg := model.User{Content: "Schizophrenia is taking me home", UserID: uint(1), ChatRoomID: uint(1)}
	// jsonBody, _ = json.Marshal(msg)
	mockService.
	On("AddUserToChatRoom", "Thom Yorke", uint(1)).
	Once().
	Return(nil)

	req = httptest.NewRequest(http.MethodPost, "/api/memberships/add-user", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	//mockService.AssertExpectations(t)
}

func TestGetChatRoomByUser(t *testing.T) {
    r := setupBasicMiddleware(t)

	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)
 
	mockService := new(MockMembershipService)
	membershipController := controller.NewMembershipController(mockService)

	membershipRoute := r.Group("/api/memberships")
	membershipRoute.Use(loadsheddingFunc)
	membershipRoute.Use(setupAuthMiddleware(t))
    {
		membershipRoute.GET("/:username/chatrooms", func(c *gin.Context) {
   	 		username := c.Param("username")  // <-- string, no conversion
    		membershipController.GetUserChatRooms(c, username)
		})
    } 

	//fail
	msg := controller.UserChatroom{Username: "DaveGrohl", ChatRoomID: uint(1)}
	jsonBody, _ := json.Marshal(msg)

	// mockService.
	// On("GetUserChatRooms", "DaveGrohl").
	// Once().
	// Return([]model.ChatRoom{model.ChatRoom{}}, errors.New("DB error"))

	mockService.
	On("GetUserSubscribedChatRooms", "DaveGrohl").
	Once().
	Return([]model.ChatRoom{model.ChatRoom{}}, errors.New("DB error"))

	req := httptest.NewRequest(http.MethodGet, "/api/memberships/DaveGrohl/chatrooms", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "error")

	//success
	// msg := model.User{Content: "Schizophrenia is taking me home", UserID: uint(1), ChatRoomID: uint(1)}
	// jsonBody, _ = json.Marshal(msg)
	// mockService.
	// On("GetUserChatRooms", "DaveGrohl").
	// Once().
	// Return([]model.ChatRoom{model.ChatRoom{}}, nil)

	mockService.
	On("GetUserSubscribedChatRooms", "DaveGrohl").
	Once().
	Return([]model.ChatRoom{model.ChatRoom{}}, nil)

	req = httptest.NewRequest(http.MethodGet, "/api/memberships/DaveGrohl/chatrooms", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	//mockService.AssertExpectations(t)
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