package service_test

import (
	"backend/internal/identity"
	"backend/internal/model"
	"testing"
	"time"
	//    "time"

	"github.com/stretchr/testify/require"
	//    "gorm.io/gorm"

	"backend/internal/repo"
	"backend/internal/service"
)

func TestChatRoomService(t *testing.T) {
	db := setupTestDB(t)
	repos := repo.NewRepoContainer(db)
	chatSvc := service.NewChatRoomService(repos)
	p := identity.New(1, "alice", "session")
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "alice", Email: "a@test.com", Password: "hash"}).Error)
	require.NoError(t, db.Create(&model.UserSession{UserID: 1, SessionID: "session", ExpiresAt: time.Now().Add(time.Hour)}).Error)

	t.Run("CreateChatRoom", func(t *testing.T) {
		// successful creation
		room, err := chatSvc.CreateChatRoom(p, "General")
		require.NoError(t, err)
		require.NotZero(t, room.ID)
		require.Equal(t, "General", room.Name)

		// duplicate creation should fail
		_, err = chatSvc.CreateChatRoom(p, "General")
		require.Error(t, err)
		require.Equal(t, "chat room already exists", err.Error())
	})

	t.Run("GetAllChatRooms", func(t *testing.T) {
		rooms, err := chatSvc.GetAllChatRooms()
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(rooms), 1)
	})

	t.Run("GetChatRoomByID", func(t *testing.T) {
		// create a new room
		room, _ := chatSvc.CreateChatRoom(p, "Tech")
		fetched, err := chatSvc.GetChatRoomByID(room.ID)
		require.NoError(t, err)
		require.Equal(t, "Tech", fetched.Name)

		// non-existing ID
		_, err = chatSvc.GetChatRoomByID(9999)
		require.Error(t, err)
		require.Equal(t, "chat room not found", err.Error())
	})

	t.Run("DeleteChatRoom", func(t *testing.T) {
		room, _ := chatSvc.CreateChatRoom(p, "Random")
		err := chatSvc.DeleteChatRoom(p, room.ID)
		require.NoError(t, err)

		// A missing room has no creator to authorize; return not found.
		err = chatSvc.DeleteChatRoom(p, room.ID)
		require.ErrorIs(t, err, service.ErrChatRoomNotFound)
	})

	t.Run("GetChatRoomByName", func(t *testing.T) {
		room, _ := chatSvc.CreateChatRoom(p, "Sports")
		fetched, err := chatSvc.GetChatRoomByName("Sports")
		require.NoError(t, err)
		require.Equal(t, room.ID, fetched.ID)

		// empty name
		_, err = chatSvc.GetChatRoomByName("")
		require.Error(t, err)
		require.Equal(t, "chat room name is required", err.Error())

		// non-existing name
		_, err = chatSvc.GetChatRoomByName("UnknownRoom")
		require.Error(t, err)
		require.Equal(t, "chat room not found", err.Error())
	})

	t.Run("SearchChatRoomsByName", func(t *testing.T) {
		// create multiple rooms
		_, _ = chatSvc.CreateChatRoom(p, "Gaming")
		_, _ = chatSvc.CreateChatRoom(p, "GameDev")
		_, _ = chatSvc.CreateChatRoom(p, "Music")

		results, err := chatSvc.SearchChatRoomsByName("Gam")
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(results), 2)

		for _, r := range results {
			require.Contains(t, r.Name, "Gam")
		}

		results, err = chatSvc.SearchChatRoomsByName("Nonexistent")
		require.NoError(t, err)
		require.Empty(t, results)
	})
}
