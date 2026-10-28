package fanout

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"fanout/internal/fanoutauth"
	kafkapb "fanout/proto/kafka"
)

type Config struct {
	GatewayPath string
	Token       string
}

type Registry interface {
	RoomUsers(context.Context, uint32) ([]uint32, error)
	UserGateways(context.Context, []uint32) (map[uint32]string, error)
}

type Dispatcher struct {
	registry Registry
	client   *http.Client
	cfg      Config
}

func NewDispatcher(reg Registry, client *http.Client, cfg Config) (*Dispatcher, error) {
	if err := fanoutauth.Validate(cfg.Token); err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("http client is required")
	}
	safeClient := *client
	safeClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Dispatcher{registry: reg, client: &safeClient, cfg: cfg}, nil
}

type FanoutRequest struct {
	RoomID  uint32              `json:"room_id"`
	UserIDs []uint32            `json:"user_ids"`
	Event   *kafkapb.KafkaEvent `json:"event"`
}

func (d *Dispatcher) Dispatch(ctx context.Context, event *kafkapb.KafkaEvent) error {
	if event == nil {
		return nil
	}

	userIDs, err := d.registry.RoomUsers(ctx, event.RoomId)
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return nil
	}

	gateways, err := d.registry.UserGateways(ctx, userIDs)
	if err != nil {
		return err
	}

	grouped := map[string][]uint32{}
	for _, userID := range userIDs {
		addr, ok := gateways[userID]
		if !ok {
			continue
		}
		grouped[addr] = append(grouped[addr], userID)
	}

	var dispatchErr error
	for addr, ids := range grouped {
		if len(ids) == 0 {
			continue
		}
		if err := d.send(ctx, addr, ids, event); err != nil {
			log.Printf("[fanout] dispatch to %s failed: %v", addr, err)
			dispatchErr = err
		}
	}

	return dispatchErr
}

func (d *Dispatcher) send(ctx context.Context, addr string, userIDs []uint32, event *kafkapb.KafkaEvent) error {
	payload, err := json.Marshal(FanoutRequest{
		RoomID:  event.RoomId,
		UserIDs: userIDs,
		Event:   event,
	})
	if err != nil {
		return fmt.Errorf("marshal fanout request: %w", err)
	}

	endpoint := d.gatewayURL(addr)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	ga := strings.TrimSpace(addr)
	if ga == "" {
		return fmt.Errorf("empty gateway address")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(fanoutauth.Header, d.cfg.Token)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// A peer may echo credentials in its body; never include it in errors/logs.
		return fmt.Errorf("gateway response status %d", resp.StatusCode)
	}

	return nil
}

func (d *Dispatcher) gatewayURL(addr string) string {
	trimmed := strings.TrimSpace(addr)
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		trimmed = "http://" + trimmed
	}

	path := d.cfg.GatewayPath
	if path == "" {
		path = "/fanout"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed + path
	}
	base := strings.TrimRight(parsed.Path, "/")
	parsed.Path = base + path
	return parsed.String()
}
