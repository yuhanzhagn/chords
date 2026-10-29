package app

import (
	"backend/internal/model"
	"fmt"
	"gorm.io/gorm"
	"time"
)

const MessageSchemaVersion uint = 2

// Check before AutoMigrate: an older binary must not silently reinterpret a newer schema.
func CheckMessageSchema(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.SchemaMigration{}) {
		return nil
	}
	var version uint
	if err := db.Model(&model.SchemaMigration{}).Select("COALESCE(MAX(version),0)").Scan(&version).Error; err != nil {
		return err
	}
	if version == 2 {
		var count int64
		if err := db.Model(&model.SchemaMigration{}).Where("version IN (1,2)").Count(&count).Error; err != nil {
			return err
		}
		if count != 2 {
			return fmt.Errorf("message schema missing migration history")
		}
		if err := validateMessageSchema(db); err != nil {
			return err
		}
		return validateV2(db)
	}
	if version == 1 {
		return validateMessageSchema(db)
	}
	if version > MessageSchemaVersion {
		return fmt.Errorf("unsupported message schema version %d", version)
	}
	return nil
}

// Each migration is explicit; v1 snapshots remain byte-for-byte unchanged.
func MigrateMessageSchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := CheckMessageSchema(tx); err != nil {
			return err
		}
		if err := tx.AutoMigrate(&model.SchemaMigration{}); err != nil {
			return err
		}
		var version uint
		if err := tx.Model(&model.SchemaMigration{}).Select("COALESCE(MAX(version),0)").Scan(&version).Error; err != nil {
			return err
		}
		if version > 0 {
			if err := validateMessageSchema(tx); err != nil {
				return err
			}
		}
		if version == 0 {
			if tx.Migrator().HasTable(&model.SendRequest{}) || tx.Migrator().HasTable(&model.MessageOutbox{}) {
				return fmt.Errorf("unversioned message tables: inspect schema before migration")
			}
			if err := tx.AutoMigrate(&model.SendRequest{}, &model.MessageOutbox{}); err != nil {
				return err
			}
			if err := tx.Create(&model.SchemaMigration{Version: 1, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}
		if version < 2 {
			if err := tx.AutoMigrate(&model.Message{}, &model.SendRequest{}, &model.MessageOutbox{}, &model.RoomSequence{}); err != nil {
				return err
			}
			// ID order is a canonical legacy order, not a reconstruction of send time.
			if err := tx.Exec(`WITH ranked AS (SELECT id, ROW_NUMBER() OVER (PARTITION BY room_id ORDER BY id) AS seq FROM messages)
    UPDATE messages SET room_seq = (SELECT seq FROM ranked WHERE ranked.id = messages.id)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`UPDATE send_requests SET room_seq = COALESCE((SELECT room_seq FROM messages WHERE messages.id = send_requests.message_id),0)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`INSERT INTO room_sequences(room_id,high_water) SELECT room_id,MAX(room_seq) FROM messages GROUP BY room_id`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE UNIQUE INDEX idx_messages_room_seq ON messages(room_id,room_seq) WHERE room_seq > 0`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE INDEX idx_outbox_due ON message_outbox(status,next_attempt,lease_until)`).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.SchemaMigration{Version: 2, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}
		return validateV2(tx)
	})
}

func validateV2(db *gorm.DB) error {
	for table, columns := range map[string][]string{"messages": {"room_seq"}, "send_requests": {"room_seq"}, "message_outbox": {"lease_token", "lease_until", "attempts", "next_attempt", "last_error"}, "room_sequences": {"room_id", "high_water"}} {
		for _, column := range columns {
			if !db.Migrator().HasColumn(table, column) {
				return fmt.Errorf("message schema missing %s.%s", table, column)
			}
		}
	}
	for table, index := range map[string]string{"messages": "idx_messages_room_seq", "message_outbox": "idx_outbox_due"} {
		if !db.Migrator().HasIndex(table, index) {
			return fmt.Errorf("message schema missing index %s", index)
		}
	}
	var indexes []struct {
		Name    string
		Unique  int
		Partial int
	}
	if err := db.Raw("PRAGMA index_list(messages)").Scan(&indexes).Error; err != nil {
		return err
	}
	valid := false
	for _, index := range indexes {
		if index.Name == "idx_messages_room_seq" && index.Unique == 1 && index.Partial == 1 {
			valid = true
		}
	}
	var columns []struct{ Name string }
	if err := db.Raw("PRAGMA index_info(idx_messages_room_seq)").Scan(&columns).Error; err != nil {
		return err
	}
	if !valid || len(columns) != 2 || columns[0].Name != "room_id" || columns[1].Name != "room_seq" {
		return fmt.Errorf("message schema invalid room sequence constraint")
	}
	return nil
}

func validateMessageSchema(db *gorm.DB) error {
	for table, columns := range map[string][]string{
		"send_requests":  {"user_id", "request_id", "fingerprint", "fingerprint_version", "room_id", "message_id", "event_id", "created_at"},
		"message_outbox": {"event_id", "room_id", "schema_version", "payload", "status", "created_at", "published_at"},
	} {
		for _, column := range columns {
			if !db.Migrator().HasColumn(table, column) {
				return fmt.Errorf("message schema missing %s.%s", table, column)
			}
		}
	}
	for table, indexes := range map[string][]string{"send_requests": {"idx_send_requests_event_id"}, "message_outbox": {"idx_outbox_pending"}} {
		for _, index := range indexes {
			if !db.Migrator().HasIndex(table, index) {
				return fmt.Errorf("message schema missing index %s", index)
			}
		}
	}
	// A corrupt/incomplete version marker must not bypass the idempotency constraint.
	var columns []struct {
		Name string
		PK   int
	}
	if err := db.Raw("PRAGMA table_info(send_requests)").Scan(&columns).Error; err != nil {
		return err
	}
	keys := map[string]int{}
	for _, column := range columns {
		if column.PK > 0 {
			keys[column.Name] = column.PK
		}
	}
	if len(keys) != 2 || keys["user_id"] != 1 || keys["request_id"] != 2 {
		return fmt.Errorf("message schema missing compound request primary key")
	}
	return nil
}
