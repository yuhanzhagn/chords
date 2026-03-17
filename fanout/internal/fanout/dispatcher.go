package fanout

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"fanout/internal/registry"
	kafkapb "fanout/proto/kafka"
)

type Config struct {
	GatewayPath string
}

type Dispatcher struct {
	registry *registry.RedisRegistry
	client   *http.Client
	cfg      Config
}

func NewDispatcher(reg *registry.RedisRegistry, client *http.Client, cfg Config) *Dispatcher {
	return &Dispatcher{registry: reg, client: client, cfg: cfg}
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

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("gateway response %s: %s", resp.Status, strings.TrimSpace(string(body)))
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
