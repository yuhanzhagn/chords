package service

import (
	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	kafkapb "backend/proto/kafka"
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type controlledProducer struct {
	publish func(context.Context, []byte, []byte) error
}

func (p controlledProducer) PublishContext(ctx context.Context, _ string, key, raw []byte) error {
	return p.publish(ctx, key, raw)
}
func (controlledProducer) Close() error { return nil }

func TestRelayRestartBrokerRecoveryLeaseAndImmutableReplay(t *testing.T) {
	db := sendDB(t)
	repos := repo.NewRepoContainer(db)
	s := NewMessageService(repos)
	result, err := s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	now := time.Now()
	var payloads [][]byte
	brokerDown := true
	producer := controlledProducer{func(ctx context.Context, key, raw []byte) error {
		require.Equal(t, []byte("1"), key)
		payloads = append(payloads, bytes.Clone(raw))
		if brokerDown {
			return errors.New("down")
		}
		return nil
	}}
	relay := NewRelay(repo.NewRepoContainer(db), producer)
	require.NoError(t, relay.Tick(context.Background(), now))
	row, err := repos.Send.FindOutbox(result.EventID)
	require.NoError(t, err)
	require.Equal(t, "pending", row.Status)
	require.Equal(t, "broker_unavailable", row.LastError)
	require.EqualValues(t, 1, row.Attempts)
	require.NoError(t, relay.Tick(context.Background(), now.Add(time.Second)))
	require.Len(t, payloads, 1, "backoff must be durable")
	brokerDown = false
	require.NoError(t, db.Exec("CREATE TRIGGER relay_fail_mark BEFORE UPDATE ON message_outbox WHEN NEW.status='published' BEGIN SELECT RAISE(ABORT,'mark failed'); END").Error)
	require.NoError(t, relay.Tick(context.Background(), now.Add(3*time.Second)))
	require.Len(t, payloads, 2)
	row, err = repos.Send.FindOutbox(result.EventID)
	require.NoError(t, err)
	require.NotEmpty(t, row.LeaseToken)
	require.Equal(t, "pending", row.Status)
	require.NoError(t, db.Exec("DROP TRIGGER relay_fail_mark").Error)
	require.NoError(t, NewRelay(repos, producer).Tick(context.Background(), now.Add(4*time.Second)))
	require.Len(t, payloads, 2)
	require.NoError(t, NewRelay(repos, producer).Tick(context.Background(), now.Add(34*time.Second)))
	require.Len(t, payloads, 3)
	for _, raw := range payloads {
		require.Equal(t, row.Payload, raw)
	}
	row, err = repos.Send.FindOutbox(result.EventID)
	require.NoError(t, err)
	require.Equal(t, "published", row.Status)
	require.NotNil(t, row.PublishedAt)
	countSendRows(t, db, 1)
}
func TestRelayQuarantineDeletionAndV1Snapshot(t *testing.T) {
	db := sendDB(t)
	repos := repo.NewRepoContainer(db)
	s := NewMessageService(repos)
	a, err := s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	require.NoError(t, s.DeleteMessage(sendPrincipal(), a.ID))
	in := sendInput()
	in.RequestID = "healthy"
	b, err := s.SendMessage(sendPrincipal(), in)
	require.NoError(t, err)
	v1, err := repos.Send.FindOutbox(b.EventID)
	require.NoError(t, err)
	var e kafkapb.KafkaEvent
	require.NoError(t, proto.Unmarshal(v1.Payload, &e))
	e.RoomSeq = 0
	old, err := proto.Marshal(&e)
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.MessageOutbox{}).Where("event_id=?", b.EventID).Updates(map[string]any{"schema_version": 1, "payload": old}).Error)
	for i, version := range []uint{1, 999} {
		require.NoError(t, db.Create(&model.MessageOutbox{EventID: fmt.Sprint("bad", i), RoomID: 1, SchemaVersion: version, Payload: []byte("bad"), Status: "pending", CreatedAt: time.Unix(0, 0)}).Error)
	}
	calls := 0
	relay := NewRelay(repos, controlledProducer{func(_ context.Context, _, raw []byte) error { calls++; require.Equal(t, old, raw); return nil }})
	require.NoError(t, relay.Tick(context.Background(), time.Now()))
	require.Equal(t, 1, calls)
	for _, id := range []string{"bad0", "bad1"} {
		row, err := repos.Send.FindOutbox(id)
		require.NoError(t, err)
		require.Equal(t, "quarantined", row.Status)
	}
	row, err := repos.Send.FindOutbox(a.EventID)
	require.NoError(t, err)
	require.Equal(t, "suppressed", row.Status)
	require.Equal(t, "message_deleted", row.LastError)
	replay, err := s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	require.True(t, replay.Deleted)
	var stored model.MessageOutbox
	require.NoError(t, db.First(&stored, "event_id=?", b.EventID).Error)
	require.Equal(t, old, stored.Payload)
}
func TestRelayOverlapCancellationAndFencing(t *testing.T) {
	db := sendDB(t)
	repos := repo.NewRepoContainer(db)
	s := NewMessageService(repos)
	result, err := s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	entered := make(chan struct{})
	var once sync.Once
	producer := controlledProducer{func(ctx context.Context, _, _ []byte) error {
		once.Do(func() { close(entered) })
		<-ctx.Done()
		return ctx.Err()
	}}
	relay := NewRelay(repos, producer)
	relay.Interval = 10 * time.Millisecond
	relay.SendTimeout = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); relay.Run(ctx) }()
	<-entered
	require.NoError(t, NewRelay(repos, producer).Tick(context.Background(), time.Now()))
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("relay failed to cancel")
	}
	row, err := repos.Send.FindOutbox(result.EventID)
	require.NoError(t, err)
	require.EqualValues(t, 1, row.Attempts)
	require.Equal(t, "publish_outcome_unknown", row.LastError)
	claimed, err := repos.ClaimOutbox(context.Background(), result.EventID, "new-owner", time.Now(), time.Second)
	require.NoError(t, err)
	require.True(t, claimed)
	require.Error(t, repos.FinishOutbox(context.Background(), result.EventID, "old-owner", "published", "", time.Now()))
	row, err = repos.Send.FindOutbox(result.EventID)
	require.NoError(t, err)
	require.Equal(t, "new-owner", row.LeaseToken)
}
func TestRoomSequencePaginationAndAuthorization(t *testing.T) {
	db := sendDB(t)
	s := NewMessageService(repo.NewRepoContainer(db))
	zero := uint64(0)
	var sent []*SendResult
	for i := 0; i < 75; i++ {
		in := sendInput()
		in.RequestID = fmt.Sprint(i)
		r, err := s.SendMessage(sendPrincipal(), in)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprint(i+1), r.RoomSeq)
		sent = append(sent, r)
	}
	for _, i := range []int{4, 29, 74} {
		require.NoError(t, s.DeleteMessage(sendPrincipal(), sent[i].ID))
	}
	page, err := s.GetSequencePage(sendPrincipal(), 1, HistoryCursor{After: &zero, Limit: 30})
	require.NoError(t, err)
	require.Equal(t, "75", page.HighWater)
	require.True(t, page.HasMore)
	snapshot := uint64(75)
	cursor := page.Messages[len(page.Messages)-1].RoomSeq
	in := sendInput()
	in.RequestID = "concurrent"
	newMessage, err := s.SendMessage(sendPrincipal(), in)
	require.NoError(t, err)
	require.Equal(t, "76", newMessage.RoomSeq)
	rows := page.Messages
	for page.HasMore {
		page, err = s.GetSequencePage(sendPrincipal(), 1, HistoryCursor{After: &cursor, Snapshot: &snapshot, Limit: 30})
		require.NoError(t, err)
		rows = append(rows, page.Messages...)
		cursor = page.Messages[len(page.Messages)-1].RoomSeq
	}
	require.Len(t, rows, 72)
	require.Equal(t, "75", page.NextCursor)
	require.EqualValues(t, 74, rows[len(rows)-1].RoomSeq)
	page, err = s.GetSequencePage(sendPrincipal(), 1, HistoryCursor{Before: &zero, Limit: 30})
	require.NoError(t, err)
	require.EqualValues(t, 76, page.Messages[len(page.Messages)-1].RoomSeq)
	before := page.Messages[0].RoomSeq
	older, err := s.GetSequencePage(sendPrincipal(), 1, HistoryCursor{Before: &before, Limit: 30})
	require.NoError(t, err)
	require.Less(t, older.Messages[len(older.Messages)-1].RoomSeq, before)
	ahead := uint64(77)
	_, err = s.GetSequencePage(sendPrincipal(), 1, HistoryCursor{After: &ahead, Limit: 30})
	require.ErrorIs(t, err, ErrCursorReset)
	_, err = s.GetSequencePage(sendPrincipal(), 1, HistoryCursor{After: &zero, Before: &zero, Limit: 30})
	require.ErrorIs(t, err, ErrInvalidCursor)
	require.NoError(t, db.Where("user_id=1").Delete(&model.UserChatRoom{}).Error)
	for _, c := range []HistoryCursor{{After: &zero, Limit: 30}, {Before: &zero, Limit: 30}} {
		_, err = s.GetSequencePage(sendPrincipal(), 1, c)
		require.ErrorIs(t, err, ErrForbidden)
	}
	require.NoError(t, db.Model(&model.UserSession{}).Where("user_id=1").Update("revoked", true).Error)
	_, err = s.GetSequencePage(identity.New(1, "1", "1"), 1, HistoryCursor{After: &zero, Limit: 30})
	require.ErrorIs(t, err, ErrUnauthenticated)
}
func TestRoomSequenceLargeIndependentRollbackAndUniqueness(t *testing.T) {
	db := sendDB(t)
	s := NewMessageService(repo.NewRepoContainer(db))
	require.NoError(t, db.Create(&model.RoomSequence{RoomID: 1, HighWater: 9007199254740992}).Error)
	r, err := s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	require.Equal(t, "9007199254740993", r.RoomSeq)
	repeated, err := s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	require.Equal(t, r.RoomSeq, repeated.RoomSeq)
	require.Error(t, db.Create(&model.Message{RoomID: 1, RoomSeq: 9007199254740993, UserID: 1, Content: "duplicate"}).Error)
	require.NoError(t, db.Create(&model.ChatRoom{ID: 2, Name: "two"}).Error)
	require.NoError(t, db.Create(&model.UserChatRoom{UserID: 1, ChatRoomID: 2}).Error)
	in := sendInput()
	in.RoomID = 2
	in.RequestID = "room-two"
	r, err = s.SendMessage(sendPrincipal(), in)
	require.NoError(t, err)
	require.Equal(t, "1", r.RoomSeq)
	require.NoError(t, db.Exec("CREATE TRIGGER seq_fail BEFORE INSERT ON message_outbox BEGIN SELECT RAISE(ABORT,'outbox');END").Error)
	in.RequestID = "rollback"
	_, err = s.SendMessage(sendPrincipal(), in)
	require.Error(t, err)
	require.NoError(t, db.Exec("DROP TRIGGER seq_fail").Error)
	r, err = s.SendMessage(sendPrincipal(), in)
	require.NoError(t, err)
	require.Equal(t, "2", r.RoomSeq)
}
func TestConcurrentDistinctRequestsAllocateUniqueRoomSequence(t *testing.T) {
	db := sendDB(t)
	s := NewMessageService(repo.NewRepoContainer(db))
	const n = 24
	results := make(chan *SendResult, n)
	failures := make(chan error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			in := sendInput()
			in.RequestID = fmt.Sprint("unique-", i)
			var r *SendResult
			var err error
			for attempt := 0; attempt < 4; attempt++ {
				r, err = s.SendMessage(sendPrincipal(), in)
				if !errors.Is(err, ErrSendBusy) {
					break
				}
			}
			results <- r
			failures <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
	sequences := map[string]bool{}
	for r := range results {
		require.NotNil(t, r)
		require.False(t, sequences[r.RoomSeq])
		sequences[r.RoomSeq] = true
	}
	for i := 1; i <= n; i++ {
		require.True(t, sequences[fmt.Sprint(i)])
	}
	countSendRows(t, db, n)
}

func TestRelayBatchClaimsUseCurrentTimeAfterSlowEarlierRecord(t *testing.T) {
	db := sendDB(t)
	repos := repo.NewRepoContainer(db)
	s := NewMessageService(repos)
	_, err := s.SendMessage(sendPrincipal(), sendInput())
	require.NoError(t, err)
	in := sendInput()
	in.RequestID = "second-in-batch"
	second, err := s.SendMessage(sendPrincipal(), in)
	require.NoError(t, err)
	calls := 0
	relay := NewRelay(repos, controlledProducer{func(_ context.Context, _, raw []byte) error {
		calls++
		if calls == 1 {
			time.Sleep(300 * time.Millisecond)
		} else {
			row, err := repos.Send.FindOutbox(second.EventID)
			require.NoError(t, err)
			require.Greater(t, row.LeaseUntil, time.Now().UnixMilli(), "later batch records need a fresh lease, not the scan start time")
		}
		return nil
	}})
	relay.Lease = 250 * time.Millisecond
	require.NoError(t, relay.Tick(context.Background(), time.Now()))
	require.Equal(t, 2, calls)
}
