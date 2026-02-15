package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	//    "gorm.io/gorm"

	"backend/internal/repo"
	"backend/internal/service"
)

func TestMessageService(t *testing.T) {
	db := setupTestDB(t)
	repos := repo.NewRepoContainer(db)
	msgSvc := service.NewMessageService(repos)

	t.Run("CreateMessage", func(t *testing.T) {
		msg, err := msgSvc.CreateMessage(1, 1, "Hello world")
		require.NoError(t, err)
		require.NotZero(t, msg.ID)
	})

	t.Run("GetMessagesByChatRoom", func(t *testing.T) {
		// create two messages for the chat room
		_, _ = msgSvc.CreateMessage(1, 100, "msg1")
		time.Sleep(2 * time.Millisecond)
		_, _ = msgSvc.CreateMessage(1, 100, "msg2")

		msgs, err := msgSvc.GetMessagesByChatRoom(100)
		require.NoError(t, err)
		require.Len(t, msgs, 2)
		require.Equal(t, "msg1", msgs[0].Content) // ensure order ascending
		require.Equal(t, "msg2", msgs[1].Content)
	})

	t.Run("GetMessagesWithLimit", func(t *testing.T) {
		// create multiple messages
		for i := 0; i < 5; i++ {
			_, _ = msgSvc.CreateMessage(1, 200, "limitMsg")
			time.Sleep(2 * time.Millisecond)
		}

		msgs, err := msgSvc.GetMessagesWithLimit(200, 3)
		require.NoError(t, err)
		require.Len(t, msgs, 3)
		// ensure descending order by CreatedAt
		require.True(t, msgs[0].CreatedAt.After(msgs[1].CreatedAt))
	})

	t.Run("DeleteMessage", func(t *testing.T) {
		msg, _ := msgSvc.CreateMessage(1, 300, "to delete")

		err := msgSvc.DeleteMessage(msg.ID)
		require.NoError(t, err)

		// deleting again should return error
		err = msgSvc.DeleteMessage(msg.ID)
		require.Error(t, err)
		require.Equal(t, "message not found", err.Error())
	})
}
