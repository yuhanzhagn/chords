package repo

import (
	"backend/internal/model"
	"context"
	"errors"
	"gorm.io/gorm"
	"math"
	"time"
)

// Called only on the same pinned transaction as message/request/outbox inserts.
func (r *RepoContainer) NextRoomSeq(room uint) (uint64, error) {
	if err := r.db.Exec("INSERT INTO room_sequences(room_id,high_water) VALUES(?,0) ON CONFLICT(room_id) DO NOTHING", room).Error; err != nil {
		return 0, err
	}
	result := r.db.Exec("UPDATE room_sequences SET high_water=high_water+1 WHERE room_id=? AND high_water < ?", room, int64(math.MaxInt64))
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected != 1 {
		return 0, errors.New("room sequence exhausted")
	}
	return r.HighWater(room)
}
func (r *RepoContainer) HighWater(room uint) (uint64, error) {
	var v uint64
	err := r.db.Model(&model.RoomSequence{}).Where("room_id=?", room).Select("high_water").Scan(&v).Error
	return v, err
}
func (r *RepoContainer) SequencePage(room uint, after, before, high uint64, limit int, descending bool) ([]model.Message, error) {
	rows := []model.Message{}
	q := r.db.Model(&model.Message{}).Select("messages.*, sr.request_id, sr.event_id").Joins("LEFT JOIN send_requests sr ON sr.message_id=messages.id AND sr.user_id=messages.user_id").Where("messages.room_id=? AND messages.room_seq > ? AND messages.room_seq <= ?", room, after, high)
	if before > 0 {
		q = q.Where("messages.room_seq < ?", before)
	}
	order := "messages.room_seq ASC"
	if descending {
		order = "messages.room_seq DESC"
	}
	err := q.Order(order).Limit(limit).Find(&rows).Error
	return rows, err
}

// Relay statements use the same pinned-connection busy cap as acceptance;
// the driver busy handler otherwise outlives cancellation at SQLite locks.
func (r *RepoContainer) relayTransaction(ctx context.Context, fn func(*RepoContainer) error) error {
	bounded, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return r.WithinTransaction(bounded, fn)
}
func (r *RepoContainer) DueOutbox(ctx context.Context, now time.Time, limit int) ([]model.MessageOutbox, error) {
	rows := []model.MessageOutbox{}
	err := r.relayTransaction(ctx, func(tx *RepoContainer) error {
		return tx.db.Where("status='pending' AND next_attempt <= ? AND lease_until <= ?", now.UnixMilli(), now.UnixMilli()).Order("created_at,event_id").Limit(limit).Find(&rows).Error
	})
	return rows, err
}
func (r *RepoContainer) ClaimOutbox(ctx context.Context, id, token string, now time.Time, lease time.Duration) (bool, error) {
	claimed := false
	err := r.relayTransaction(ctx, func(tx *RepoContainer) error {
		result := tx.db.Model(&model.MessageOutbox{}).Where("event_id=? AND status='pending' AND lease_until <= ?", id, now.UnixMilli()).Updates(map[string]any{"lease_token": token, "lease_until": now.Add(lease).UnixMilli(), "attempts": gorm.Expr("attempts+1")})
		claimed = result.RowsAffected == 1
		return result.Error
	})
	return claimed, err
}
func (r *RepoContainer) FinishOutbox(ctx context.Context, id, token, status, reason string, next time.Time) error {
	return r.relayTransaction(ctx, func(tx *RepoContainer) error {
		fields := map[string]any{"status": status, "last_error": reason, "next_attempt": next.UnixMilli(), "lease_token": "", "lease_until": 0}
		if status == "published" {
			fields["published_at"] = time.Now().UTC()
		}
		result := tx.db.Model(&model.MessageOutbox{}).Where("event_id=? AND status='pending' AND lease_token=?", id, token).Updates(fields)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("outbox lease lost")
		}
		return nil
	})
}
func (r *RepoContainer) ReadClaimedOutbox(ctx context.Context, id string) (*model.MessageOutbox, error) {
	var row *model.MessageOutbox
	err := r.relayTransaction(ctx, func(tx *RepoContainer) error { var err error; row, err = tx.Send.FindOutbox(id); return err })
	return row, err
}
func (r *RepoContainer) OutboxMessageExists(ctx context.Context, id uint64) (bool, error) {
	var count int64
	err := r.relayTransaction(ctx, func(tx *RepoContainer) error {
		return tx.db.Model(&model.Message{}).Where("id=?", id).Count(&count).Error
	})
	return count > 0, err
}
