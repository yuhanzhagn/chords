package source

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"connection/internal/gateway"
	kafkapb "connection/proto/kafka"
)

const maxFanoutBodyBytes = 1 << 20

type FanoutRequest struct {
	RoomID  uint32              `json:"room_id"`
	UserIDs []uint32            `json:"user_ids"`
	Event   *kafkapb.KafkaEvent `json:"event"`
}

func NewFanoutHTTPHandler(hub *gateway.Hub[*kafkapb.KafkaEvent]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxFanoutBodyBytes))
		if err != nil {
			http.Error(w, "failed to read request", http.StatusBadRequest)
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("[fanout-http] failed to close body: %v", err)
			}
		}()

		var req FanoutRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Event == nil {
			http.Error(w, "event is required", http.StatusBadRequest)
			return
		}
		if err := applyFanout(hub, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}

func applyFanout(hub *gateway.Hub[*kafkapb.KafkaEvent], req *FanoutRequest) error {
	if hub == nil {
		return errors.New("hub is required")
	}
	if req == nil || req.Event == nil {
		return errors.New("event is required")
	}

	payload, err := hub.Codec().Encode(req.Event)
	if err != nil {
		log.Printf("[fanout-http] encode error: %v", err)
		return errors.New("failed to encode event")
	}

	if len(req.UserIDs) == 0 {
		hub.Broadcast(req.Event.RoomId, payload)
		return nil
	}

	hub.SendToClients(req.UserIDs, payload)
	return nil
}
