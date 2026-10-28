package service_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	//    "gorm.io/gorm"

	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

func TestMessageService(t *testing.T) {
	db := setupTestDB(t)
	repos := repo.NewRepoContainer(db)
	msgSvc := service.NewMessageService(repos)
	p := identity.New(1, "alice", "session")
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "alice", Email: "a@test.com", Password: "hash"}).Error)
	require.NoError(t, db.Create(&model.UserSession{UserID: 1, SessionID: "session", ExpiresAt: time.Now().Add(time.Hour)}).Error)
	for _, id := range []uint{1, 100, 200, 300} {
		require.NoError(t, db.Create(&model.ChatRoom{ID: id, Name: fmt.Sprint(id)}).Error)
		require.NoError(t, db.Create(&model.UserChatRoom{UserID: 1, ChatRoomID: id}).Error)
	}

	t.Run("CreateMessage", func(t *testing.T) {
		msg, err := msgSvc.CreateMessage(p, 1, 1, "Hello world")
		require.NoError(t, err)
		require.NotZero(t, msg.ID)
	})

	t.Run("GetMessagesByChatRoom", func(t *testing.T) {
		// create two messages for the chat room
		_, _ = msgSvc.CreateMessage(p, 1, 100, "msg1")
		time.Sleep(2 * time.Millisecond)
		_, _ = msgSvc.CreateMessage(p, 1, 100, "msg2")

		msgs, err := msgSvc.GetMessagesByChatRoom(p, 100)
		require.NoError(t, err)
		require.Len(t, msgs, 2)
		require.Equal(t, "msg1", msgs[0].Content) // ensure order ascending
		require.Equal(t, "msg2", msgs[1].Content)
	})

	t.Run("GetMessagesWithLimit", func(t *testing.T) {
		// create multiple messages
		for i := 0; i < 5; i++ {
			_, _ = msgSvc.CreateMessage(p, 1, 200, "limitMsg")
			time.Sleep(2 * time.Millisecond)
		}

		msgs, err := msgSvc.GetMessagesWithLimit(p, 200, 3)
		require.NoError(t, err)
		require.Len(t, msgs, 3)
		// ensure descending order by CreatedAt
		require.True(t, msgs[0].CreatedAt.After(msgs[1].CreatedAt))
	})

	t.Run("DeleteMessage", func(t *testing.T) {
		msg, _ := msgSvc.CreateMessage(p, 1, 300, "to delete")

		err := msgSvc.DeleteMessage(p, msg.ID)
		require.NoError(t, err)

		// deleting again should return error
		err = msgSvc.DeleteMessage(p, msg.ID)
		require.Error(t, err)
		require.Equal(t, "message not found", err.Error())
	})
}
