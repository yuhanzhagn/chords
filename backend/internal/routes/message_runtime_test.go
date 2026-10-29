package routes

import (
	"backend/internal/app"
	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestHTTPAssemblyAndRelayStopWithUnavailableBroker(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "runtime.sqlite")+"?_pragma=busy_timeout(5000)"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()
	require.NoError(t, app.InitDB(db))
	// A closed loopback port is an unavailable broker, no external infrastructure.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	listener.Close()
	t.Setenv("CHORDS_KAFKA_BROKERS", address)
	t.Setenv("CHORDS_LEGACY_COMMANDS", "")
	require.NotNil(t, SetupRouter(db, nil), "router construction must never connect to Kafka")
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "runtime", Email: "runtime@test", Password: "hash"}).Error)
	require.NoError(t, db.Create(&model.UserSession{UserID: 1, SessionID: "runtime", ExpiresAt: time.Now().Add(time.Hour)}).Error)
	require.NoError(t, db.Create(&model.ChatRoom{ID: 1, Name: "runtime"}).Error)
	require.NoError(t, db.Create(&model.UserChatRoom{UserID: 1, ChatRoomID: 1}).Error)
	stop := StartMessageRuntime(context.Background(), db)
	s := service.NewMessageService(repo.NewRepoContainer(db))
	receipt, err := s.SendMessage(identity.New(1, "runtime", "runtime"), service.SendInput{UserID: 1, RoomID: 1, Content: "broker down", RequestID: "runtime"})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		var out model.MessageOutbox
		if db.First(&out, "event_id=?", receipt.EventID).Error != nil {
			return false
		}
		return out.Attempts > 0
	}, 3*time.Second, 10*time.Millisecond)
	started := time.Now()
	stop()
	require.Less(t, time.Since(started), 3*time.Second)
	var out model.MessageOutbox
	require.NoError(t, db.First(&out, "event_id=?", receipt.EventID).Error)
	require.Equal(t, "pending", out.Status)
}
