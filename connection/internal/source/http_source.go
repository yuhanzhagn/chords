package source

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"connection/internal/gateway"
	kafkapb "connection/proto/kafka"
)

const maxFanoutBodyBytes = 1 << 20

type FanoutRequest struct {
	RoomID  uint32              `json:"room_id"`
	UserIDs []uint32            `json:"user_ids"`
	Event   *kafkapb.KafkaEvent `json:"event"`
}

type FanoutHTTPSource struct {
	hub     *gateway.Hub[*kafkapb.KafkaEvent]
	address string
	server  *http.Server
}

func NewFanoutHTTPHandler(hub *gateway.Hub[*kafkapb.KafkaEvent], address string) *FanoutHTTPSource {
	return &FanoutHTTPSource{hub: hub, address: address}
}

func (s *FanoutHTTPSource) Start(_ context.Context) error {
	if s.server != nil {
		return errors.New("fanout http source already started")
	}

	mux := http.NewServeMux()
	mux.Handle("/fanout", s)
	s.server = &http.Server{
		Addr:              s.address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[fanout-http] server error: %v", err)
		}
	}()

	return nil
}

func (s *FanoutHTTPSource) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	err := s.server.Shutdown(ctx)
	s.server = nil
	return err
}

func (s *FanoutHTTPSource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	if err := applyFanout(s.hub, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[fanout-http] room=%d users=%d msg_type=%s", req.Event.RoomId, len(req.UserIDs), req.Event.MsgType)
	w.WriteHeader(http.StatusNoContent)
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

	if req.RoomID != 0 {
		hub.Broadcast(req.RoomID, payload)
		return nil
	}
	if len(req.UserIDs) == 0 {
		return errors.New("room_id or user_ids is required")
	}
	hub.SendToClients(req.UserIDs, payload)
	return nil
}
