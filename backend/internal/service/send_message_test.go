package service

import (
	"backend/internal/app"
	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	kafkapb "backend/proto/kafka"
	"backend/utils"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func sendDB(t *testing.T) *gorm.DB { t.Helper(); return sendDBTimeout(t, 10) }

func sendDBTimeout(t *testing.T, busyMillis int) *gorm.DB {
	t.Helper()
	return sendDBOptions(t, busyMillis, "WAL")
}

func sendDBOptions(t *testing.T, busyMillis int, journal string) *gorm.DB {
	t.Helper()
	// A real file, WAL and multiple connections exercise SQLite's writer/upgrade races.
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "send.sqlite")+fmt.Sprintf("?_pragma=busy_timeout(%d)&_pragma=foreign_keys(1)&_pragma=journal_mode(%s)", busyMillis, journal)), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { sqlDB.Close() })
	require.NoError(t, app.InitDB(db))
	for i := uint(1); i <= 2; i++ {
		require.NoError(t, db.Create(&model.User{ID: i, Username: fmt.Sprint(i), Email: fmt.Sprintf("%d@test", i), Password: "hash"}).Error)
		require.NoError(t, db.Create(&model.UserSession{UserID: i, SessionID: fmt.Sprint(i), ExpiresAt: time.Now().Add(time.Hour)}).Error)
	}
	require.NoError(t, db.Create(&model.ChatRoom{ID: 1, Name: "room"}).Error)
	for i := uint(1); i <= 2; i++ {
		require.NoError(t, db.Create(&model.UserChatRoom{UserID: i, ChatRoomID: 1}).Error)
	}
	return db
}
func countSendRows(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()
	for _, table := range []any{&model.Message{}, &model.SendRequest{}, &model.MessageOutbox{}} {
		var n int64
		require.NoError(t, db.Model(table).Count(&n).Error)
		require.Equal(t, want, n, fmt.Sprintf("%T", table))
	}
}
func sendPrincipal() identity.Principal { return identity.New(1, "1", "1") }
func sendInput() SendInput {
	return SendInput{UserID: 1, RoomID: 1, Content: "你好\x00\\\"", RequestID: "request", TempID: "pending", ClientID: 123}
}

func TestSendAtomicIdempotencyAndTombstone(t *testing.T) {
	db := sendDB(t)
	s := NewMessageService(repo.NewRepoContainer(db))
	in := sendInput()
	p := sendPrincipal()
	a, err := s.SendMessage(p, in)
	require.NoError(t, err)
	b, err := s.SendMessage(p, in)
	require.NoError(t, err)
	require.True(t, b.Replayed)
	require.Equal(t, a.EventID, b.EventID)
	require.Equal(t, a.ServerMessageID, b.ServerMessageID)
	require.Equal(t, a.PersistedAt, b.PersistedAt)
	countSendRows(t, db, 1)
	changed := in
	changed.Content += "different"
	_, err = s.SendMessage(p, changed)
	require.ErrorIs(t, err, ErrRequestConflict)
	changed = in
	changed.RoomID = 2
	require.NoError(t, db.Create(&model.ChatRoom{ID: 2, Name: "second"}).Error)
	require.NoError(t, db.Create(&model.UserChatRoom{UserID: 1, ChatRoomID: 2}).Error)
	_, err = s.SendMessage(p, changed)
	require.ErrorIs(t, err, ErrRequestConflict)
	// New login, same user/key, retains original facts; session isn't part of the unique key.
	require.NoError(t, db.Model(&model.UserSession{}).Where("user_id = ?", 1).Update("revoked", true).Error)
	require.NoError(t, db.Create(&model.UserSession{UserID: 1, SessionID: "new-login", ExpiresAt: time.Now().Add(time.Hour)}).Error)
	_, err = s.SendMessage(p, in)
	require.ErrorIs(t, err, ErrUnauthenticated)
	p = identity.New(1, "1", "new-login")
	b, err = s.SendMessage(p, in)
	require.NoError(t, err)
	require.Equal(t, a.EventID, b.EventID)
	require.NoError(t, s.DeleteMessage(p, a.ID))
	b, err = s.SendMessage(p, in)
	require.NoError(t, err)
	require.True(t, b.Deleted)
	require.Equal(t, a.EventID, b.EventID)
	require.Equal(t, a.PersistedAt, b.PersistedAt)
	var messages int64
	require.NoError(t, db.Model(&model.Message{}).Count(&messages).Error)
	require.Zero(t, messages)
	// Another user with the same key owns an independent namespace.
	in.UserID = 2
	b, err = s.SendMessage(identity.New(2, "2", "2"), in)
	require.NoError(t, err)
	require.NotEqual(t, a.EventID, b.EventID)
	var requests int64
	require.NoError(t, db.Model(&model.SendRequest{}).Count(&requests).Error)
	require.EqualValues(t, 2, requests)
}

func TestSendConcurrentSQLiteRequests(t *testing.T) {
	db := sendDB(t)
	s := NewMessageService(repo.NewRepoContainer(db))
	const n = 24
	results := make(chan *SendResult, n)
	errs := make(chan error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			var r *SendResult
			var err error
			for retry := 0; retry < 4; retry++ {
				r, err = s.SendMessage(sendPrincipal(), sendInput())
				if !errors.Is(err, ErrSendBusy) {
					break
				}
			}
			results <- r
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var first *SendResult
	for r := range results {
		if first == nil {
			first = r
		}
		require.Equal(t, first.EventID, r.EventID)
		require.Equal(t, first.ServerMessageID, r.ServerMessageID)
		require.Equal(t, first.PersistedAt, r.PersistedAt)
		require.Equal(t, first.RoomSeq, r.RoomSeq)
		require.Equal(t, "1", r.RoomSeq)
	}
	countSendRows(t, db, 1)
	// Direct duplicate inserts prove the database enforces the compound key.
	var row model.SendRequest
	require.NoError(t, db.First(&row).Error)
	row.EventID = "other-event"
	require.Error(t, db.Create(&row).Error)
}

func TestSendRollsBackEveryInsertAndCommit(t *testing.T) {
	for _, failure := range []string{"messages", "send_requests", "message_outbox", "commit"} {
		t.Run(failure, func(t *testing.T) {
			db := sendDB(t)
			s := NewMessageService(repo.NewRepoContainer(db))
			if failure == "commit" {
				// A deferred FK violation is only detected by SQLite's actual COMMIT, after all inserts.
				require.NoError(t, db.Exec("CREATE TABLE fault_parent(id integer PRIMARY KEY)").Error)
				require.NoError(t, db.Exec("CREATE TABLE fault_child(id integer REFERENCES fault_parent(id) DEFERRABLE INITIALLY DEFERRED)").Error)
				require.NoError(t, db.Exec("CREATE TRIGGER inject_failure AFTER INSERT ON message_outbox BEGIN INSERT INTO fault_child VALUES(1); END").Error)
			} else {
				require.NoError(t, db.Exec("CREATE TRIGGER inject_failure BEFORE INSERT ON "+failure+" BEGIN SELECT RAISE(ABORT,'injected insert failure'); END").Error)
			}
			_, err := s.SendMessage(sendPrincipal(), sendInput())
			require.Error(t, err)
			countSendRows(t, db, 0)
			require.NoError(t, db.Exec("DROP TRIGGER inject_failure").Error)
			_, err = s.SendMessage(sendPrincipal(), sendInput())
			require.NoError(t, err)
			countSendRows(t, db, 1)
		})
	}
}

type sendProducer struct {
	payloads [][]byte
	keys     [][]byte
	err      error
}

func (p *sendProducer) Publish(topic string, key, raw []byte) error {
	if topic != "notification" {
		return errors.New("wrong topic")
	}
	p.payloads = append(p.payloads, bytes.Clone(raw))
	p.keys = append(p.keys, bytes.Clone(key))
	return p.err
}
func (*sendProducer) Close() error { return nil }

func TestSendOutboxPublishAndMarkFailures(t *testing.T) {
	db := sendDB(t)
	producer := &sendProducer{err: errors.New("broker unavailable")}
	s := NewMessageService(repo.NewRepoContainer(db), producer)
	a, err := s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	countSendRows(t, db, 1)
	var out model.MessageOutbox
	require.NoError(t, db.First(&out).Error)
	require.Equal(t, "pending", out.Status)
	require.Equal(t, out.Payload, producer.payloads[0])
	require.Equal(t, []byte("1"), producer.keys[0])
	var event kafkapb.KafkaEvent
	require.NoError(t, proto.Unmarshal(out.Payload, &event))
	require.Equal(t, a.EventID, event.EventId)
	require.Equal(t, a.PersistedAt, event.PersistedAt)
	require.Equal(t, sendInput().TempID, event.TempId)
	require.NotContains(t, string(out.Payload), "new-login")
	producer.err = nil
	require.NoError(t, db.Exec("CREATE TRIGGER fail_mark BEFORE UPDATE ON message_outbox WHEN NEW.status='published' BEGIN SELECT RAISE(ABORT,'mark failed'); END").Error)
	b, err := s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	require.Equal(t, a.EventID, b.EventID)
	require.NoError(t, db.First(&out).Error)
	require.Equal(t, "pending", out.Status)
	require.NoError(t, db.Exec("DROP TRIGGER fail_mark").Error)
	// Failed publication mark keeps the claim until lease expiry, then replays the same bytes.
	require.NoError(t, db.Model(&model.MessageOutbox{}).Where("event_id=?", a.EventID).Update("lease_until", 0).Error)
	b, err = s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	require.Equal(t, a.EventID, b.EventID)
	require.Len(t, producer.payloads, 3)
	for _, payload := range producer.payloads {
		require.Equal(t, out.Payload, payload)
	}
	require.NoError(t, db.First(&out).Error)
	require.Equal(t, "published", out.Status)
	require.NotNil(t, out.PublishedAt)
	_, err = s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	require.Len(t, producer.payloads, 3)
}

func TestSendCurrentAuthorizationAndBusy(t *testing.T) {
	db := sendDB(t)
	s := NewMessageService(repo.NewRepoContainer(db))
	_, err := s.SendMessage(identity.Principal{}, sendInput())
	require.Error(t, err)
	in := sendInput()
	in.UserID = 2
	_, err = s.SendMessage(sendPrincipal(), in)
	require.ErrorIs(t, err, ErrForbidden)
	countSendRows(t, db, 0)
	_, err = s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	require.NoError(t, db.Where("user_id = ?", 1).Delete(&model.UserChatRoom{}).Error)
	_, err = s.SendMessage(sendPrincipal(), sendInput())
	require.ErrorIs(t, err, ErrForbidden)
	require.NoError(t, db.Create(&model.UserChatRoom{UserID: 1, ChatRoomID: 1}).Error)
	require.NoError(t, db.Model(&model.UserSession{}).Where("user_id = ?", 1).Update("revoked", true).Error)
	_, err = s.SendMessage(sendPrincipal(), sendInput())
	require.ErrorIs(t, err, ErrUnauthenticated)
	countSendRows(t, db, 1)
	require.NoError(t, db.Model(&model.UserSession{}).Where("user_id = ?", 1).Update("revoked", false).Error)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	require.NoError(t, tx.Exec("UPDATE chat_rooms SET name=name WHERE id=1").Error)
	start := time.Now()
	in = sendInput()
	in.RequestID = "locked"
	_, err = s.SendMessage(sendPrincipal(), in)
	require.ErrorIs(t, err, ErrSendBusy)
	require.Less(t, time.Since(start), 3*time.Second)
	require.NoError(t, tx.Rollback().Error)
	countSendRows(t, db, 1)
}

func TestSendLegacyIDs(t *testing.T) {
	db := sendDB(t)
	s := NewMessageService(repo.NewRepoContainer(db))
	in := sendInput()
	in.RequestID = ""
	a, err := s.SendMessage(sendPrincipal(), in)
	require.NoError(t, err)
	require.Equal(t, in.TempID, a.RequestID)
	require.True(t, a.Deduplicated)
	b, err := s.SendMessage(sendPrincipal(), in)
	require.NoError(t, err)
	require.Equal(t, a.EventID, b.EventID)
	in.TempID = ""
	a, err = s.SendMessage(sendPrincipal(), in)
	require.NoError(t, err)
	require.False(t, a.Deduplicated)
	require.Len(t, a.RequestID, 32)
	b, err = s.SendMessage(sendPrincipal(), in)
	require.NoError(t, err)
	require.NotEqual(t, a.RequestID, b.RequestID)
}

func TestKafkaAndRESTShareCommittedRequestAndSignature(t *testing.T) {
	db := sendDB(t)
	producer := &sendProducer{err: errors.New("broker down")}
	s := NewMessageService(repo.NewRepoContainer(db), producer)
	in := sendInput()
	a, err := s.SendMessage(sendPrincipal(), in)
	require.NoError(t, err)
	event := &kafkapb.KafkaEvent{Id: in.ClientID, UserId: uint32(in.UserID), RoomId: uint32(in.RoomID), MsgType: "message", Content: []byte(in.Content), TempId: in.TempID, RequestId: in.RequestID}
	raw, err := proto.Marshal(event)
	require.NoError(t, err)
	grant, err := utils.SignCommand(utils.CommandClaims{UserID: 1, Username: "1", SessionID: "1", RoomID: 1, Digest: utils.EventDigest(raw)})
	require.NoError(t, err)
	ks := KafkaService{MessageService: s}
	// New fields are part of the command's actual transported digest.
	altered := proto.Clone(event).(*kafkapb.KafkaEvent)
	altered.RequestId = "forged-request"
	ks.HandleOutboundEvent(altered, grant)
	require.Len(t, producer.payloads, 1)
	producer.err = nil
	ks.HandleOutboundEvent(event, grant)
	countSendRows(t, db, 1)
	require.Len(t, producer.payloads, 2)
	require.Equal(t, producer.payloads[0], producer.payloads[1])
	var out model.MessageOutbox
	require.NoError(t, db.First(&out).Error)
	var published kafkapb.KafkaEvent
	require.NoError(t, proto.Unmarshal(out.Payload, &published))
	require.Equal(t, a.EventID, published.EventId)
	// Credentials are carried only in the inbound header, never in persisted or outbound fields.
	require.NotContains(t, string(out.Payload), grant)
	require.Equal(t, in.Content, string(published.Content))
}

func TestSendReceiptAndHistoryPreserveLargeIDs(t *testing.T) {
	db := sendDB(t)
	s := NewMessageService(repo.NewRepoContainer(db))
	require.NoError(t, db.Create(&model.Message{ID: 9007199254740992, UserID: 1, RoomID: 1, Content: "legacy"}).Error)
	result, err := s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	require.Equal(t, "9007199254740993", result.ServerMessageID)
	raw, err := json.Marshal(result)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"server_message_id":"9007199254740993"`)
	messages, err := s.GetMessagesByChatRoom(sendPrincipal(), 1)
	require.NoError(t, err)
	raw, err = json.Marshal(messages)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"server_message_id":"9007199254740993"`)
	var out model.MessageOutbox
	require.NoError(t, db.First(&out).Error)
	var event kafkapb.KafkaEvent
	require.NoError(t, proto.Unmarshal(out.Payload, &event))
	require.Equal(t, uint64(9007199254740993), event.ServerMessageId)
}

func TestSendBusyWithLongDriverTimeout(t *testing.T) {
	db := sendDBTimeout(t, 5000)
	s := NewMessageService(repo.NewRepoContainer(db))
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	require.NoError(t, tx.Exec("UPDATE chat_rooms SET name=name WHERE id=1").Error)
	start := time.Now()
	_, err := s.SendMessage(sendPrincipal(), sendInput())
	require.ErrorIs(t, err, ErrSendBusy)
	require.Less(t, time.Since(start), 3*time.Second)
	require.NoError(t, tx.Rollback().Error)
	countSendRows(t, db, 0)
}

func TestSendCommitBusyIsBoundedAndRolledBack(t *testing.T) {
	db := sendDBOptions(t, 5000, "DELETE")
	s := NewMessageService(repo.NewRepoContainer(db))
	reader := db.Begin()
	require.NoError(t, reader.Error)
	defer reader.Rollback()
	var n int64
	require.NoError(t, reader.Model(&model.Message{}).Count(&n).Error)
	start := time.Now()
	_, err := s.SendMessage(sendPrincipal(), sendInput())
	require.ErrorIs(t, err, ErrSendBusy)
	require.Less(t, time.Since(start), 3*time.Second)
	require.NoError(t, reader.Rollback().Error)
	countSendRows(t, db, 0)
	_, err = s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	countSendRows(t, db, 1)
}
