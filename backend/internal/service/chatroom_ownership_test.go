package service_test

import (
	"testing"
	"time"

	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
	"github.com/stretchr/testify/require"
)

func TestChatRoomMutationsRecheckSessionAndUnknownOwnership(t *testing.T) {
	db := setupTestDB(t)
	repos := repo.NewRepoContainer(db)
	rooms := service.NewChatRoomService(repos)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "alice", Email: "a@test.com", Password: "hash"}).Error)
	session := model.UserSession{UserID: 1, SessionID: "owner-session", ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, db.Create(&session).Error)
	p := identity.New(1, "alice", session.SessionID)
	owned, err := rooms.CreateChatRoom(p, "owned")
	require.NoError(t, err)
	_, err = rooms.CreateChatRoom(identity.Principal{}, "anonymous")
	require.ErrorIs(t, err, service.ErrUnauthenticated)
	require.ErrorIs(t, rooms.DeleteChatRoom(identity.Principal{}, owned.ID), service.ErrUnauthenticated)
	zero := uint(0)
	for _, owner := range []*uint{nil, &zero} {
		room := model.ChatRoom{Name: time.Now().String(), CreatorID: owner}
		require.NoError(t, db.Create(&room).Error)
		require.ErrorIs(t, rooms.DeleteChatRoom(p, room.ID), service.ErrForbidden)
	}
	// A write must retain its own ownership predicate, beyond the service lookup.
	rows, err := repos.ChatRoom.DeleteOwned(owned.ID, 2)
	require.NoError(t, err)
	require.Zero(t, rows)
	for _, update := range []map[string]any{{"revoked": true}, {"revoked": false, "expires_at": time.Now().Add(-time.Hour)}} {
		require.NoError(t, db.Model(&session).Updates(update).Error)
		_, err = rooms.CreateChatRoom(p, "stale-session")
		require.ErrorIs(t, err, service.ErrUnauthenticated)
		require.ErrorIs(t, rooms.DeleteChatRoom(p, owned.ID), service.ErrUnauthenticated)
		require.NoError(t, db.First(&model.ChatRoom{}, owned.ID).Error)
	}
	// A database error must not bypass authorization or delete the room.
	require.NoError(t, db.Migrator().DropTable(&model.UserSession{}))
	require.Error(t, rooms.DeleteChatRoom(p, owned.ID))
	require.NoError(t, db.First(&model.ChatRoom{}, owned.ID).Error)
}

// Simulate an offline reassignment between the service lookup and the delete.
// The repository must still enforce creator_id on the actual write.
type reassignedRoomRepo struct {
	repo.ChatRoomRepo
	reassign func(uint) error
}

func (r reassignedRoomRepo) GetByID(id uint) (*model.ChatRoom, error) {
	room, err := r.ChatRoomRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if err := r.reassign(id); err != nil {
		return nil, err
	}
	return room, nil
}

func TestChatRoomDeletionChecksOwnershipAtWrite(t *testing.T) {
	db := setupTestDB(t)
	repos := repo.NewRepoContainer(db)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "alice", Email: "a@test.com", Password: "hash"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "bob", Email: "b@test.com", Password: "hash"}).Error)
	require.NoError(t, db.Create(&model.UserSession{UserID: 1, SessionID: "session", ExpiresAt: time.Now().Add(time.Hour)}).Error)
	p := identity.New(1, "alice", "session")
	rooms := service.NewChatRoomService(repos)
	room, err := rooms.CreateChatRoom(p, "reassigned")
	require.NoError(t, err)
	repos.ChatRoom = reassignedRoomRepo{ChatRoomRepo: repos.ChatRoom, reassign: func(id uint) error {
		return db.Model(&model.ChatRoom{}).Where("id = ?", id).Update("creator_id", 2).Error
	}}
	require.ErrorIs(t, rooms.DeleteChatRoom(p, room.ID), service.ErrForbidden)
	var stored model.ChatRoom
	require.NoError(t, db.First(&stored, room.ID).Error)
	require.NotNil(t, stored.CreatorID)
	require.Equal(t, uint(2), *stored.CreatorID)
}
