package main

import (
	"bytes"
	"connection/internal/handler"
	kafkapb "connection/proto/kafka"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (a *backendAuthorizer) accept(c *handler.Context, token string, event *kafkapb.KafkaEvent) error {
	// Only the backend's authorized, committed receipt can produce persisted.
	body, _ := json.Marshal(map[string]any{"user_id": event.UserId, "chat_room_id": event.RoomId, "content": string(event.Content), "request_id": event.RequestId, "temp_id": event.TempId})
	req, err := http.NewRequestWithContext(c.Context, http.MethodPost, strings.TrimRight(a.url, "/")+"/api/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	response, err := a.client.Do(req)
	status := 503
	if err == nil {
		defer response.Body.Close()
		status = response.StatusCode
	}
	reply := kafkapb.KafkaEvent{MsgType: "send_error", UserId: event.UserId, RoomId: event.RoomId, RequestId: event.RequestId, TempId: event.TempId}
	outcome, retryable := "unknown", true
	switch status {
	case 400, 401, 403, 409, 413:
		outcome, retryable = "rejected", false
	}
	if status != 200 && status != 201 && status != 400 && status != 401 && status != 403 && status != 409 && status != 413 && status != 429 && status < 500 {
		status = 503
	}
	if err == nil && (status == 200 || status == 201) {
		var receipt struct {
			UserID    uint32 `json:"UserID"`
			RoomID    uint32 `json:"RoomID"`
			RequestID string `json:"request_id"`
			EventID   string `json:"event_id"`
			ID        string `json:"server_message_id"`
			Time      string `json:"persisted_at"`
			Seq       string `json:"room_seq"`
			Deleted   bool   `json:"deleted"`
		}
		if json.NewDecoder(io.LimitReader(response.Body, kafkapb.MaxRESTBodyBytes)).Decode(&receipt) == nil {
			id, e1 := strconv.ParseUint(receipt.ID, 10, 64)
			seq, e2 := strconv.ParseUint(receipt.Seq, 10, 63)
			_, timeErr := time.Parse(time.RFC3339Nano, receipt.Time)
			expectedRequest := event.RequestId
			if expectedRequest == "" {
				expectedRequest = event.TempId
			}
			if e1 == nil && e2 == nil && id > 0 && (seq > 0 || receipt.Deleted) && receipt.EventID != "" && receipt.RequestID != "" && timeErr == nil && receipt.UserID == event.UserId && receipt.RoomID == event.RoomId && (expectedRequest == "" || expectedRequest == receipt.RequestID) {
				reply.MsgType = "persisted"
				if receipt.Deleted {
					reply.MsgType = "deleted"
				}
				reply.RequestId = receipt.RequestID
				reply.EventId = receipt.EventID
				reply.ServerMessageId = id
				reply.RoomSeq = seq
				reply.PersistedAt = receipt.Time
				reply.Content = event.Content
			} else {
				status = 503
				outcome = "unknown"
				retryable = true
			}
		} else {
			status = 503
			outcome = "unknown"
			retryable = true
		}
	}
	if reply.MsgType == "send_error" {
		reply.Content, _ = json.Marshal(map[string]any{"code": status, "outcome": outcome, "retryable": retryable})
	}
	if c.Reply == nil {
		return nil
	}
	return c.Reply(&reply)
}
