package routes

import (
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"backend/internal/cache"
	"backend/internal/controller"
	"backend/internal/logrus"
	"backend/internal/middleware/jwtauth"
	"backend/internal/middleware/loadshedding"
	"backend/internal/middleware/logger"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
	"backend/internal/websocket"
)

/*
func SetupWebsocket(r *gin.Engine){
    r.GET("/ws/:roomID/:userID", func(c *gin.Context) {
        roomID, _ := strconv.ParseUint(c.Param("roomID"), 10, 64)
        userID, _ := strconv.ParseUint(c.Param("userID"), 10, 64)
        websocket.ServeWs(uint(roomID), uint(userID), c.Writer, c.Request)
    })
}*/

func SetupAuthRouter(r *gin.RouterGroup, authService service.AuthService, loadsheddingFunc gin.HandlerFunc){
	// authService := service.NewAuthService(db, typedCache)
	authController := controller.NewAuthController(authService)

	authRoutes := r.Group("/auth")
	authRoutes.Use(loadsheddingFunc)
    {
        authRoutes.POST("/register", authController.Register)
        authRoutes.POST("/login", authController.Login)
		authRoutes.POST("/logout", authController.Logout)
    } 
}

func SetupMembershipRouter(r *gin.RouterGroup, s service.MembershipService, authFunc gin.HandlerFunc, loadsheddingFunc gin.HandlerFunc){
   // membershipService := service.NewMembershipService(db)
    membershipController := controller.NewMembershipController(s)
	
	memberships := r.Group("/memberships")

	memberships.Use(loadsheddingFunc)
    // Apply middleware to all /chatrooms routes
    memberships.Use(authFunc)	

	{
        memberships.POST("/add-user", membershipController.AddUser)
		//memberships.POST("/chatrooms", membershipController.GetUserChatRooms)
		memberships.GET("/:username/chatrooms", func(c *gin.Context) {
   	 		username := c.Param("username")  // <-- string, no conversion
    		membershipController.GetUserChatRooms(c, username)
		})

    }

}


func SetupMessageRouter(r *gin.RouterGroup, messageService service.MessageService, authFunc gin.HandlerFunc, loadsheddingFunc gin.HandlerFunc) {
	messageController := controller.NewMessageController(messageService)

	r.POST("/messages", loadsheddingFunc, authFunc, messageController.CreateMessage)
	r.GET("/chatrooms/:id/messages", loadsheddingFunc, authFunc, messageController.GetMessagesByChatRoom)
	r.DELETE("/messages/:id", loadsheddingFunc, authFunc, messageController.DeleteMessage)
	r.GET("/ws/:userID", func(c *gin.Context) {
		userID, _ := strconv.ParseUint(c.Param("userID"), 10, 64)
		websocket.ServeWs(uint(userID), c.Writer, c.Request, messageService)
	})
}

func SetupChatroomRouter(r *gin.RouterGroup, chatRoomService service.ChatRoomService, authFunc gin.HandlerFunc, loadsheddingFunc gin.HandlerFunc) {
	chatRoomController := controller.NewChatRoomController(chatRoomService)

	chatrooms := r.Group("/chatrooms")
	chatrooms.Use(loadsheddingFunc)
	chatrooms.Use(authFunc)
	{
		chatrooms.POST("", chatRoomController.CreateChatRoom)
		chatrooms.GET("", chatRoomController.GetAllChatRooms)
		chatrooms.GET("/:id", chatRoomController.GetChatRoomByID)
		chatrooms.DELETE("/:id", chatRoomController.DeleteChatRoom)
		chatrooms.GET("/search", chatRoomController.SearchChatRooms)
	}
}

func SetupUserRouter(r *gin.RouterGroup, userService service.UserService, authFunc gin.HandlerFunc, loadsheddingFunc gin.HandlerFunc) {
	userController := controller.NewUserController(userService)

	users := r.Group("/users")
	users.Use(loadsheddingFunc)
	users.Use(authFunc)
	{
		users.POST("", userController.CreateUser)
		users.GET("", userController.GetUsers)
	}
}


func SetupRouter(db *gorm.DB, rds *redis.Client) *gin.Engine{
	//r := gin.Default()
    r := gin.New()
	// init logger
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

	typedCache := cache.NewTypedCache[service.BlockEntry](10*time.Minute, 15*time.Minute)
	redisCache := cache.NewRedisCache[[]model.ChatRoom](rds)

	repos := repo.NewRepoContainer(db)

	authService := service.NewAuthService(repos, typedCache)
	userService := service.NewUserService(repos)
	chatRoomService := service.NewChatRoomService(repos)
	membershipService := service.NewMembershipService(repos, redisCache)
	messageService := service.NewMessageService(repos)

	authFunc := jwtauth.NewAuthMiddleware(authService).Auth()
	loadsheddingFunc := loadshedding.LoadShedding(20, 5, 100*time.Millisecond)

	api := r.Group("/api")
	SetupUserRouter(api, userService, authFunc, loadsheddingFunc)
	SetupChatroomRouter(api, chatRoomService, authFunc, loadsheddingFunc)
	SetupMessageRouter(api, messageService, authFunc, loadsheddingFunc)
	SetupAuthRouter(api, authService, loadsheddingFunc)
	SetupMembershipRouter(api, membershipService, authFunc, loadsheddingFunc)
	return r
}
