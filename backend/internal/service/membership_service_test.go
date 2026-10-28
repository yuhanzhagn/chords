package service_test

import (
	"backend/internal/cache"
	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func setupCache() *cache.TypedCache[[]model.ChatRoom] {
	return cache.NewTypedCache[[]model.ChatRoom](10*time.Minute, 15*time.Minute)
}
func TestMembershipReadsRequireCaller(t *testing.T) {
	db := setupTestDB(t)
	repos := repo.NewRepoContainer(db)
	svc := service.NewMembershipService(repos, setupCache())
	user := model.User{Username: "alice", Email: "a@test.com", Password: "hash"}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.UserSession{UserID: user.ID, SessionID: "session", ExpiresAt: time.Now().Add(time.Hour)}).Error)
	p := identity.New(user.ID, user.Username, "session")
	_, err := svc.GetUserSubscribedChatRooms(identity.Principal{}, "alice")
	require.ErrorIs(t, err, service.ErrUnauthenticated)
	for _, username := range []string{"", "ghost", "bob"} {
		_, err = svc.GetUserSubscribedChatRooms(p, username)
		require.ErrorIs(t, err, service.ErrForbidden)
	}
	rooms, err := svc.GetUserSubscribedChatRooms(p, "alice")
	require.NoError(t, err)
	require.Empty(t, rooms)
	room := model.ChatRoom{Name: "General"}
	require.NoError(t, db.Create(&room).Error)
	require.NoError(t, svc.AddUserToChatRoom(p, "alice", room.ID))
	rooms, err = svc.GetUserSubscribedChatRooms(p, "alice")
	require.NoError(t, err)
	require.Len(t, rooms, 1)
	require.Equal(t, room.ID, rooms[0].ID)
	require.NoError(t, repos.UserSession.RevokeAllByUserID(user.ID))
	_, err = svc.GetUserSubscribedChatRooms(p, "alice")
	require.ErrorIs(t, err, service.ErrUnauthenticated)
}
