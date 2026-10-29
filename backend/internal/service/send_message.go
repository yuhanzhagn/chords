package service

import (
	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	kafkapb "backend/proto/kafka"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"math"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

var ErrRequestConflict = errors.New("request_id already used for different content")
var ErrSendBusy = errors.New("message store busy; retry with the same request_id")
var errRequestRace = errors.New("concurrent request committed; retry transaction")

type SendInput struct {
	UserID, RoomID    uint
	Content           string
	RequestID, TempID string
	ClientID          uint64
}

type SendResult struct {
	// Preserve the existing REST message fields during additive protocol migration.
	ID              uint
	Content         string `json:",omitempty"`
	UserID          uint
	RoomID          uint
	CreatedAt       time.Time
	RequestID       string `json:"request_id"`
	EventID         string `json:"event_id"`
	ServerMessageID string `json:"server_message_id"`
	RoomSeq         string `json:"room_seq"`
	PersistedAt     string `json:"persisted_at"`
	Deleted         bool   `json:"deleted"`
	Replayed        bool   `json:"replayed"`
	Deduplicated    bool   `json:"deduplication_supported"`
}

func randomSendID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func (s *messageService) SendMessage(p identity.Principal, in SendInput) (*SendResult, error) {
	if in.UserID != p.UserID() {
		return nil, ErrForbidden
	}
	if in.RoomID == 0 || in.RoomID > math.MaxUint32 || in.UserID > math.MaxUint32 {
		return nil, kafkapb.ErrInvalidMessage
	}
	incoming := &kafkapb.KafkaEvent{Id: in.ClientID, UserId: uint32(in.UserID), RoomId: uint32(in.RoomID), MsgType: "message", Content: []byte(in.Content), RequestId: in.RequestID, TempId: in.TempID}
	if err := kafkapb.ValidateSend(incoming); err != nil {
		return nil, err
	}
	dedup := in.RequestID != "" || in.TempID != ""
	if in.RequestID == "" {
		in.RequestID = in.TempID
	}
	if in.RequestID == "" {
		var err error
		in.RequestID, err = randomSendID()
		if err != nil {
			return nil, err
		}
	}
	// JSON array is unambiguous for arbitrary UTF-8 content. Persist exactly these bytes, without normalization.
	fingerprintInput, _ := json.Marshal([]any{1, in.RoomID, in.Content})
	hash := sha256.Sum256(fingerprintInput)
	fingerprint := hex.EncodeToString(hash[:])
	eventID, err := randomSendID()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var result *SendResult
	for attempt := 0; attempt < 12; attempt++ {
		err = s.repos.WithinTransaction(ctx, func(tx *repo.RepoContainer) error {
			// Reads and writes share a SQLite snapshot. A concurrent revocation/write causes a busy
			// upgrade or serializes after this commit; retries recheck all authorization.
			if err := requireMember(tx, p, in.RoomID); err != nil {
				return err
			}
			request, err := tx.Send.FindRequest(p.UserID(), in.RequestID)
			if err == nil {
				if request.FingerprintVersion != 1 || request.Fingerprint != fingerprint {
					return ErrRequestConflict
				}
				msg, readErr := tx.Message.GetByID(request.MessageID)
				if readErr != nil && !errors.Is(readErr, gorm.ErrRecordNotFound) {
					return readErr
				}
				result = sendReceipt(request, msg, errors.Is(readErr, gorm.ErrRecordNotFound), true, dedup)
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			seq, err := tx.NextRoomSeq(in.RoomID)
			if err != nil {
				return err
			}
			msg := &model.Message{RoomSeq: seq, UserID: p.UserID(), RoomID: in.RoomID, Content: in.Content, CreatedAt: time.Now().UTC()}
			if err := tx.Message.Create(msg); err != nil {
				return err
			}
			request = &model.SendRequest{UserID: p.UserID(), RequestID: in.RequestID, Fingerprint: fingerprint, FingerprintVersion: 1, RoomID: in.RoomID, MessageID: msg.ID, RoomSeq: msg.RoomSeq, EventID: eventID, CreatedAt: msg.CreatedAt}
			inserted, err := tx.Send.InsertRequest(request)
			if err != nil {
				return err
			}
			if !inserted {
				return errRequestRace
			}
			event := &kafkapb.KafkaEvent{Id: in.ClientID, UserId: uint32(msg.UserID), RoomId: uint32(msg.RoomID), MsgType: "message", Content: []byte(msg.Content), TempId: in.TempID,
				RequestId: in.RequestID, EventId: eventID, ServerMessageId: uint64(msg.ID), CreatedAt: msg.CreatedAt.Unix(), PersistedAt: msg.CreatedAt.Format(time.RFC3339Nano), RoomSeq: msg.RoomSeq}
			payload, err := proto.Marshal(event)
			if err != nil {
				return err
			}
			if err := tx.Send.InsertOutbox(&model.MessageOutbox{EventID: eventID, RoomID: msg.RoomID, SchemaVersion: 2, Payload: payload, Status: "pending", CreatedAt: msg.CreatedAt}); err != nil {
				return err
			}
			result = sendReceipt(request, msg, false, false, dedup)
			return nil
		})
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			return nil, ErrSendBusy
		}
		if !sqliteBusy(err) && !errors.Is(err, errRequestRace) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ErrSendBusy
		case <-time.After(time.Duration(attempt+1) * 5 * time.Millisecond):
		}
	}
	if err != nil {
		return nil, ErrSendBusy
	}
	if !result.Deleted && s.producer != nil {
		if err := s.PublishPending(result.EventID, s.producer); err != nil {
			log.Println("message persisted; outbox publish remains pending")
		}
	}
	return result, nil
}

func sqliteBusy(err error) bool {
	var code interface{ Code() int }
	return errors.As(err, &code) && (code.Code()&255 == 5 || code.Code()&255 == 6)
}
func sendReceipt(r *model.SendRequest, m *model.Message, deleted, replayed, dedup bool) *SendResult {
	result := &SendResult{ID: r.MessageID, RoomSeq: strconv.FormatUint(r.RoomSeq, 10), UserID: r.UserID, RoomID: r.RoomID, CreatedAt: r.CreatedAt, RequestID: r.RequestID, EventID: r.EventID, ServerMessageID: strconv.FormatUint(uint64(r.MessageID), 10), PersistedAt: r.CreatedAt.UTC().Format(time.RFC3339Nano), Deleted: deleted, Replayed: replayed, Deduplicated: dedup}
	if !deleted {
		result.Content = m.Content
	}
	return result
}
