package service_test

import (
//    "errors"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
//    "gorm.io/gorm"

    "backend/internal/model"
    "backend/internal/repo"
    "backend/internal/service"
	"backend/internal/cache"
)

func setupCache() *cache.TypedCache[[]model.ChatRoom]{
	return cache.NewTypedCache[[]model.ChatRoom](10 * time.Minute, 15 * time.Minute)
}

func TestGetUserChatRoomsFromDB(t *testing.T) {
    db := setupTestDB(t)
    cache := setupCache()
    repos := repo.NewRepoContainer(db)
    svc := service.NewMembershipService(repos, cache)

    t.Run("empty username", func(t *testing.T) {
        rooms, err := svc.GetUserChatRoomsFromDB("")
        require.Error(t, err)
        require.Nil(t, rooms)
    })

    t.Run("user not found", func(t *testing.T) {
        rooms, err := svc.GetUserChatRoomsFromDB("ghost")
        require.NoError(t, err)
        require.Nil(t, rooms)
    })

    t.Run("user exists but no rooms", func(t *testing.T) {
        user := model.User{Username: "alice",  Email:"alice@test.com", Password:"test123"}
        require.NoError(t, db.Create(&user).Error)

        rooms, err := svc.GetUserChatRoomsFromDB("alice")
        require.NoError(t, err)
        require.Empty(t, rooms)
    })

    t.Run("user with rooms", func(t *testing.T) {
        user := model.User{Username: "bob", Email:"bob@test.com", Password:"test123"}
        require.NoError(t, db.Create(&user).Error)

        room := model.ChatRoom{Name: "General"}
        require.NoError(t, db.Create(&room).Error)

        err := db.Create(&model.UserChatRoom{
            UserID:     user.ID,
            ChatRoomID: room.ID,
            JoinedAt:   time.Now(),
        }).Error
		require.NoError(t, err)

        rooms, err := svc.GetUserChatRoomsFromDB("bob")
        require.NoError(t, err)
        require.Len(t, rooms, 1)
        require.Equal(t, "General", rooms[0].Name)
    })
}

