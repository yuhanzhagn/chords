package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	kafkaadapter "connection/internal/adapter/kafka"
	"connection/internal/gateway"
	"connection/internal/handler"
	"connection/internal/handler/middlewares"
	kafkapb "connection/proto/kafka"
	"google.golang.org/protobuf/proto"
)

type authGrant struct {
	UserID       uint32 `json:"user_id"`
	Username     string `json:"username"`
	SessionID    string `json:"session_id"`
	CommandToken string `json:"command_token"`
}
type backendAuthorizer struct {
	url    string
	client *http.Client
}

func (a *backendAuthorizer) authorize(ctx context.Context, token string, roomID uint32, kind, digest string) (authGrant, error) {
	body, err := json.Marshal(struct {
		RoomID uint32 `json:"room_id"`
		Kind   string `json:"kind"`
		Digest string `json:"digest"`
	}{roomID, kind, digest})
	if err != nil {
		return authGrant{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(a.url, "/")+"/api/auth/ws-authorize", bytes.NewReader(body))
	if err != nil {
		return authGrant{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	response, err := a.client.Do(req)
	if err != nil {
		return authGrant{}, gateway.ErrAuthUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode >= 500 || response.StatusCode == 429 {
		return authGrant{}, gateway.ErrAuthUnavailable
	}
	if response.StatusCode != http.StatusOK {
		return authGrant{}, fmt.Errorf("authorization rejected (%d)", response.StatusCode)
	}
	var grant authGrant
	if err := json.NewDecoder(io.LimitReader(response.Body, 8192)).Decode(&grant); err != nil {
		return authGrant{}, err
	}
	if grant.UserID == 0 || grant.Username == "" || grant.SessionID == "" {
		return authGrant{}, errors.New("incomplete principal")
	}
	return grant, nil
}
func (a *backendAuthorizer) legacyAuthenticator(next handler.HandlerFunc) gateway.Authenticator {
	return a.buildAuthenticator(next, false)
}
func (a *backendAuthorizer) authenticator(next handler.HandlerFunc) gateway.Authenticator {
	return a.buildAuthenticator(next, true)
}
func (a *backendAuthorizer) buildAuthenticator(next handler.HandlerFunc, synchronous bool) gateway.Authenticator {
	limit := middlewares.ConnectionRateLimitMiddleware(middlewares.ConnectionRateLimitOptions{RatePerSecond: 20, Burst: 40, IdleTTL: 5 * time.Minute})
	return func(ctx context.Context, raw []byte) (gateway.AuthenticatedSession, error) {
		var frame struct {
			Type  string `json:"type"`
			Token string `json:"token"`
		}
		if err := json.Unmarshal(raw, &frame); err != nil || frame.Type != "auth" || frame.Token == "" {
			return gateway.AuthenticatedSession{}, errors.New("authentication required")
		}
		principal, err := a.authorize(ctx, frame.Token, 0, "", "")
		if err != nil {
			return gateway.AuthenticatedSession{}, err
		}
		verify := func(ctx context.Context, room uint32, kind, digest string) (authGrant, error) {
			grant, err := a.authorize(ctx, frame.Token, room, kind, digest)
			if err != nil {
				return authGrant{}, err
			}
			if grant.UserID != principal.UserID || grant.Username != principal.Username || grant.SessionID != principal.SessionID {
				return authGrant{}, errors.New("principal changed")
			}
			return grant, nil
		}
		return gateway.AuthenticatedSession{
			UserID:          principal.UserID,
			ValidateSession: func() error { _, err := verify(context.Background(), 0, "", ""); return err },
			AuthorizeDelivery: func(room uint32) error {
				kind := "subscribe"
				if room == 0 {
					kind = ""
				}
				_, err := verify(context.Background(), room, kind, "")
				return err
			},
			Handler: limit(func(c *handler.Context) error {
				inbound, ok := c.Event.(gateway.InboundEvent[*kafkapb.KafkaEvent])
				if !ok || inbound.Event == nil {
					return gateway.ErrPolicyViolation
				}
				event := inbound.Event
				if event.UserId != 0 && event.UserId != principal.UserID {
					return gateway.ErrPolicyViolation
				}
				if event.RoomId == 0 || (event.MsgType != "join" && event.MsgType != "leave" && event.MsgType != "message") {
					return gateway.ErrPolicyViolation
				}
				event.UserId = principal.UserID
				if event.MsgType == "message" {
					if err := kafkapb.ValidateSend(event); err != nil {
						if errors.Is(err, kafkapb.ErrMessageTooLarge) {
							return gateway.ErrMessageTooLarge
						}
						return gateway.ErrPolicyViolation
					}
				}
				if synchronous && event.MsgType == "message" {
					return a.accept(c, frame.Token, event)
				}
				raw, err := proto.Marshal(event)
				if err != nil {
					return gateway.ErrPolicyViolation
				}
				hash := sha256.Sum256(raw)
				grant, err := verify(c.Context, event.RoomId, event.MsgType, hex.EncodeToString(hash[:]))
				if err != nil {
					if errors.Is(err, gateway.ErrAuthUnavailable) {
						return err
					}
					return gateway.ErrPolicyViolation
				}
				if event.MsgType == "message" && grant.CommandToken == "" {
					return gateway.ErrPolicyViolation
				}
				if len(grant.CommandToken) > kafkapb.MaxCommandHeaderBytes {
					return gateway.ErrMessageTooLarge
				}
				c.Context = kafkaadapter.WithCommandToken(c.Context, grant.CommandToken)
				return next(c)
			}),
		}, nil
	}
}
func newBackendAuthorizer(url string) *backendAuthorizer {
	return &backendAuthorizer{url: url, client: &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
