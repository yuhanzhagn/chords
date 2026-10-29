package app_test

import (
	"backend/internal/app"
	"backend/internal/model"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMessageSchemaUpgradeAndReadCompatibility(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{true: "legacy", false: "fresh"}[legacy], func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migration.sqlite")), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { sqlDB.Close() })
			if legacy {
				require.NoError(t, db.AutoMigrate(&model.Message{}))
				require.NoError(t, db.Create(&model.Message{ID: 17, UserID: 7, RoomID: 4, Content: "old", CreatedAt: time.Unix(1000, 0).UTC()}).Error)
			}
			require.NoError(t, app.InitDB(db))
			require.NoError(t, app.InitDB(db))
			var version model.SchemaMigration
			require.NoError(t, db.Order("version desc").First(&version).Error)
			require.Equal(t, app.MessageSchemaVersion, version.Version)
			for _, table := range []any{&model.SendRequest{}, &model.MessageOutbox{}} {
				var n int64
				require.NoError(t, db.Model(table).Count(&n).Error)
				require.Zero(t, n, "old messages must never cause notification replay")
			}
			// Legacy model still reads the same table and columns; extra tables are additive.
			if legacy {
				var old struct {
					ID        uint
					UserID    uint
					RoomID    uint
					Content   string
					CreatedAt time.Time
				}
				require.NoError(t, db.Table("messages").First(&old).Error)
				require.Equal(t, uint(17), old.ID)
				require.Equal(t, "old", old.Content)
				require.Equal(t, time.Unix(1000, 0).UTC(), old.CreatedAt.UTC())
			}
			require.NoError(t, db.Create(&model.SchemaMigration{Version: app.MessageSchemaVersion + 1, AppliedAt: time.Now()}).Error)
			require.ErrorContains(t, app.InitDB(db), "unsupported message schema")
		})
	}
}

func TestMessageSchemaRejectsUnversionedOrIncompleteTables(t *testing.T) {
	for _, incomplete := range []bool{false, true} {
		t.Run(map[bool]string{false: "unversioned", true: "incomplete"}[incomplete], func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "bad-schema.sqlite")), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { sqlDB.Close() })
			if incomplete {
				require.NoError(t, app.InitDB(db))
				require.NoError(t, db.Migrator().DropIndex(&model.MessageOutbox{}, "idx_outbox_pending"))
			} else {
				require.NoError(t, db.AutoMigrate(&model.SendRequest{}))
			}
			require.Error(t, app.InitDB(db))
		})
	}
}

func TestExplicitV1ToV2PreservesFactsAndSnapshots(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "v1.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()
	// Exact v1 DDL, deliberately independent of the current model definitions.
	for _, ddl := range []string{
		`CREATE TABLE messages(id integer PRIMARY KEY AUTOINCREMENT, content text NOT NULL,user_id integer NOT NULL,room_id integer NOT NULL,created_at datetime)`,
		`CREATE TABLE schema_migrations(version integer PRIMARY KEY,applied_at datetime NOT NULL)`,
		`CREATE TABLE send_requests(user_id integer NOT NULL,request_id text NOT NULL,fingerprint text NOT NULL,fingerprint_version integer NOT NULL,room_id integer NOT NULL,message_id integer NOT NULL,event_id text NOT NULL,created_at datetime,PRIMARY KEY(user_id,request_id))`,
		`CREATE UNIQUE INDEX idx_send_requests_event_id ON send_requests(event_id)`,
		`CREATE TABLE message_outbox(event_id text PRIMARY KEY,room_id integer NOT NULL,schema_version integer NOT NULL,payload blob NOT NULL,status text NOT NULL,created_at datetime,published_at datetime)`,
		`CREATE INDEX idx_outbox_pending ON message_outbox(status)`,
		`INSERT INTO schema_migrations VALUES(1,'2026-01-01')`,
		`INSERT INTO messages VALUES(20,'later-id',7,4,'2020-01-01'),(10,'earlier-id',7,4,'2020-01-01'),(30,'other-room',7,5,'2010-01-01')`,
		`INSERT INTO send_requests VALUES(7,'present','fp',1,4,20,'event-present','2020-01-01'),(7,'deleted','fp2',1,4,99,'event-deleted','2020-01-01')`,
		`INSERT INTO message_outbox VALUES('event-present',4,1,x'0102ff','pending','2020-01-01',NULL),('event-deleted',4,1,x'abcdef','published','2020-01-01','2020-01-02')`,
	} {
		require.NoError(t, db.Exec(ddl).Error)
	}
	require.NoError(t, app.InitDB(db))
	require.NoError(t, app.InitDB(db))
	var messages []model.Message
	require.NoError(t, db.Order("id").Find(&messages).Error)
	require.EqualValues(t, 1, messages[0].RoomSeq)
	require.EqualValues(t, 2, messages[1].RoomSeq)
	require.EqualValues(t, 1, messages[2].RoomSeq)
	require.Equal(t, "earlier-id", messages[0].Content)
	require.Equal(t, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), messages[0].CreatedAt.UTC())
	var requests []model.SendRequest
	require.NoError(t, db.Order("request_id").Find(&requests).Error)
	require.Len(t, requests, 2)
	require.EqualValues(t, 0, requests[0].RoomSeq, "deleted old request remains a tombstone, not a forged message")
	require.EqualValues(t, 2, requests[1].RoomSeq)
	var out []model.MessageOutbox
	require.NoError(t, db.Order("event_id").Find(&out).Error)
	require.Len(t, out, 2)
	require.Equal(t, []byte{0xab, 0xcd, 0xef}, out[0].Payload)
	require.Equal(t, "published", out[0].Status)
	require.NotNil(t, out[0].PublishedAt)
	require.Equal(t, []byte{1, 2, 255}, out[1].Payload)
	require.Equal(t, "pending", out[1].Status)
	require.EqualValues(t, 1, out[1].SchemaVersion)
	var high model.RoomSequence
	require.NoError(t, db.First(&high, "room_id=4").Error)
	require.EqualValues(t, 2, high.HighWater)
	require.Error(t, db.Create(&model.Message{RoomID: 4, RoomSeq: 2, UserID: 7, Content: "dup"}).Error)
	// Read-only old query still works; old writers/relay must stay stopped on rollback.
	var legacy struct {
		ID      uint
		Content string
	}
	require.NoError(t, db.Table("messages").First(&legacy, "id=20").Error)
	require.Equal(t, "later-id", legacy.Content)
}
func TestV2MigrationFailureRollsBackAndCorruptionRefusesStartup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "rollback.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()
	require.NoError(t, db.Exec(`CREATE TABLE messages(id integer PRIMARY KEY,content text NOT NULL,user_id integer NOT NULL,room_id integer NOT NULL,created_at datetime)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE schema_migrations(version integer PRIMARY KEY,applied_at datetime NOT NULL)`).Error)
	require.NoError(t, db.Exec(`CREATE TRIGGER migration_fail BEFORE INSERT ON schema_migrations WHEN NEW.version=2 BEGIN SELECT RAISE(ABORT,'v2 failed');END`).Error)
	require.ErrorContains(t, app.InitDB(db), "v2 failed")
	require.False(t, db.Migrator().HasColumn("messages", "room_seq"))
	require.False(t, db.Migrator().HasTable(&model.RoomSequence{}))
	require.False(t, db.Migrator().HasTable(&model.SendRequest{}))
	require.NoError(t, db.Exec("DROP TRIGGER migration_fail").Error)
	require.NoError(t, app.InitDB(db))
	require.NoError(t, db.Exec("DROP INDEX idx_messages_room_seq").Error)
	require.ErrorContains(t, app.InitDB(db), "missing index")
	require.False(t, db.Migrator().HasIndex("messages", "idx_messages_room_seq"), "startup must not silently repair a damaged versioned schema")
}

func TestV2RejectsMisleadingIndexOrMissingMigrationHistory(t *testing.T) {
	for _, fault := range []string{"nonunique-index", "missing-v1-marker"} {
		t.Run(fault, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "corrupt.sqlite")), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			defer sqlDB.Close()
			require.NoError(t, app.InitDB(db))
			if fault == "nonunique-index" {
				require.NoError(t, db.Exec("DROP INDEX idx_messages_room_seq").Error)
				require.NoError(t, db.Exec("CREATE INDEX idx_messages_room_seq ON messages(room_id,room_seq) WHERE room_seq>0").Error)
			} else {
				require.NoError(t, db.Where("version=1").Delete(&model.SchemaMigration{}).Error)
			}
			require.Error(t, app.InitDB(db))
		})
	}
}
