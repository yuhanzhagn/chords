package service

import (
	"backend/internal/repo"
	kafkapb "backend/proto/kafka"
	"context"
	"errors"
	"log"
	"math"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"
)

// ContextProducer is the production relay contract. Publish must honor cancellation;
// the legacy MessageProducer adapter remains only for injected synchronous callers.
type ContextProducer interface {
	PublishContext(context.Context, string, []byte, []byte) error
	Close() error
}
type Relay struct {
	Repos                        *repo.RepoContainer
	Producer                     ContextProducer
	Interval, Lease, SendTimeout time.Duration
	Batch                        int
}

func NewRelay(r *repo.RepoContainer, p ContextProducer) *Relay {
	return &Relay{r, p, time.Second, 30 * time.Second, 5 * time.Second, 32}
}
func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		if err := r.Tick(ctx, time.Now()); err != nil && ctx.Err() == nil {
			log.Println("outbox scan failed")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (r *Relay) Tick(ctx context.Context, now time.Time) error {
	started := time.Now()
	rows, err := r.Repos.DueOutbox(ctx, now, r.Batch)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := r.publish(ctx, row.EventID, now.Add(time.Since(started))); err != nil && ctx.Err() == nil {
			log.Printf("outbox event=%s remains pending", row.EventID)
		}
	}
	return nil
}
func (r *Relay) publish(ctx context.Context, id string, now time.Time) error {
	started := time.Now()
	token, err := randomSendID()
	if err != nil {
		return err
	}
	claimed, err := r.Repos.ClaimOutbox(ctx, id, token, now, r.Lease)
	if err != nil || !claimed {
		return err
	}
	row, err := r.Repos.ReadClaimedOutbox(ctx, id)
	if err != nil {
		return err
	}
	finish := func(status, reason string, next time.Time) error {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		return r.Repos.FinishOutbox(cleanup, id, token, status, reason, next)
	}
	var event kafkapb.KafkaEvent
	if (row.SchemaVersion != 1 && row.SchemaVersion != 2) || proto.Unmarshal(row.Payload, &event) != nil || event.EventId != row.EventID || uint(event.RoomId) != row.RoomID || event.ServerMessageId == 0 || event.ServerMessageId > math.MaxInt64 || event.RoomId == 0 || event.UserId == 0 || event.RoomSeq > math.MaxInt64 || event.MsgType != "message" || kafkapb.ValidateSend(&event) != nil || (row.SchemaVersion == 2 && event.RoomSeq == 0) {
		log.Printf("outbox event=%s quarantined: invalid snapshot/version", id)
		return finish("quarantined", "invalid_snapshot_or_version", now)
	}
	exists, err := r.Repos.OutboxMessageExists(ctx, event.ServerMessageId)
	if err != nil {
		return err
	}
	if !exists {
		return finish("suppressed", "message_deleted", now)
	}
	sendCtx, cancel := context.WithTimeout(ctx, r.SendTimeout)
	if r.Producer == nil {
		err = errors.New("producer unavailable")
	} else {
		err = r.Producer.PublishContext(sendCtx, "notification", []byte(strconv.FormatUint(uint64(row.RoomID), 10)), row.Payload)
	}
	cancel()
	if err != nil {
		shift := row.Attempts
		if shift > 6 {
			shift = 6
		}
		reason := "broker_unavailable"
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			reason = "publish_outcome_unknown"
		}
		markErr := finish("pending", reason, now.Add(time.Since(started)).Add(time.Second*time.Duration(1<<shift)))
		return errors.Join(err, markErr)
	}
	// A failure here leaves the lease intact. Expiry replays identical bytes/event_id.
	return finish("published", "", now)
}

type legacyProducer struct{ MessageProducer }

func (p legacyProducer) PublishContext(_ context.Context, topic string, key, value []byte) error {
	return p.Publish(topic, key, value)
}
func (s *messageService) PublishPending(id string, p MessageProducer) error {
	if p == nil {
		return errors.New("producer unavailable")
	}
	row, err := s.repos.Send.FindOutbox(id)
	if err != nil {
		return err
	}
	if row.Status != "pending" {
		return nil
	}
	return NewRelay(s.repos, legacyProducer{p}).publish(context.Background(), id, time.Now())
}
