package app_test

import (
	"path/filepath"
	"testing"
	"time"

	"backend/internal/app"
	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestChatRoomOwnershipSchemaCompatibility(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "fresh"
		if legacy {
			name = "legacy"
		}
		t.Run(name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "schema.sqlite")), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			defer sqlDB.Close()
			if legacy {
				require.NoError(t, db.Exec("CREATE TABLE chat_rooms (id integer PRIMARY KEY AUTOINCREMENT, name text NOT NULL UNIQUE, created_at datetime)").Error)
				require.NoError(t, db.Exec("INSERT INTO chat_rooms(id,name,created_at) VALUES(17,'legacy-room','2020-01-02 03:04:05+00:00')").Error)
				require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.UserChatRoom{}))
				require.NoError(t, db.Create(&model.User{ID: 7, Username: "first-member", Email: "first@test.com", Password: "hash"}).Error)
				require.NoError(t, db.Create(&model.UserChatRoom{UserID: 7, ChatRoomID: 17}).Error)
			}
			// The real initialization path adds a nullable indexed column, never assigns owners.
			require.NoError(t, app.InitDB(db))
			require.True(t, db.Migrator().HasColumn(&model.ChatRoom{}, "CreatorID"))
			require.True(t, db.Migrator().HasIndex(&model.ChatRoom{}, "idx_chat_rooms_creator_id"))
			if !legacy {
				require.NoError(t, db.Create(&model.User{ID: 7, Username: "first-member", Email: "first@test.com", Password: "hash"}).Error)
			}
			require.NoError(t, db.Create(&model.UserSession{UserID: 7, SessionID: "schema-test-session", ExpiresAt: time.Now().Add(time.Hour)}).Error)
			p := identity.New(7, "first-member", "schema-test-session")
			rooms := service.NewChatRoomService(repo.NewRepoContainer(db))
			if legacy {
				var old model.ChatRoom
				require.NoError(t, db.First(&old, 17).Error)
				require.Nil(t, old.CreatorID)
				require.Equal(t, "legacy-room", old.Name)
				require.Equal(t, time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC), old.CreatedAt.UTC())
				require.ErrorIs(t, rooms.DeleteChatRoom(p, old.ID), service.ErrForbidden)
				var count int64
				require.NoError(t, db.Model(&model.UserChatRoom{}).Where("chat_room_id = ?", 17).Count(&count).Error)
				require.EqualValues(t, 1, count)
			}
			created, err := rooms.CreateChatRoom(p, "new-room")
			require.NoError(t, err)
			require.NotNil(t, created.CreatorID)
			require.Equal(t, uint(7), *created.CreatorID)
			// Re-running migration must preserve both unknown and known ownership.
			require.NoError(t, app.InitDB(db))
			var newRoom model.ChatRoom
			require.NoError(t, db.First(&newRoom, created.ID).Error)
			require.Equal(t, created.CreatorID, newRoom.CreatorID)
			if legacy {
				var old model.ChatRoom
				require.NoError(t, db.First(&old, 17).Error)
				require.Nil(t, old.CreatorID)
			}
			require.NoError(t, rooms.DeleteChatRoom(p, created.ID))
		})
	}
}
