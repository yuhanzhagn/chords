package main

import (
	kafkaadapter "connection/internal/adapter/kafka"
	"connection/internal/app"
	"connection/internal/event/codec"
	"connection/internal/gateway"
	"connection/internal/handler"
	"connection/internal/handler/middlewares"
	"connection/internal/sink"
	"connection/internal/source"
	kafkapb "connection/proto/kafka"
	"context"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const devSharedJWTSecret = "dev-shared-jwt-secret"

var nextWSClientID atomic.Uint32

type ConnectionJWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func WsHandler[T any](hub *gateway.Hub[T], inboundHandler handler.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := nextWSClientID.Add(1)
		if userID == 0 {
			userID = nextWSClientID.Add(1)
		}
		gateway.ServeWs(userID, w, r, hub, inboundHandler)
	})
}

func messageEventSinkWriter(
	hub *gateway.Hub[*kafkapb.KafkaEvent],
	multiSink sink.Sink[*kafkapb.KafkaEvent],
) handler.SinkFunc {
	return func(ctx context.Context, event any) error {
		inbound, ok := event.(gateway.InboundEvent[*kafkapb.KafkaEvent])
		if !ok {
			return fmt.Errorf("unexpected event type: %T", event)
		}
		log.Printf("[ws-inbound] client=%d room=%d msg_type=%s", inbound.ClientID, inbound.Event.RoomId, inbound.Event.MsgType)
		if !hub.IsMessage(inbound.Event) {
			return nil
		}
		if err := multiSink.Write(ctx, inbound.Event); err != nil {
			return fmt.Errorf("send inbound event to sinks: %w", err)
		}
		return nil
	}
}

func groupAssignmentHandler(hub *gateway.Hub[*kafkapb.KafkaEvent]) handler.HandlerFunc {
	return func(c *handler.Context) error {
		inbound, ok := c.Event.(gateway.InboundEvent[*kafkapb.KafkaEvent])
		if !ok {
			return fmt.Errorf("unexpected event type: %T", c.Event)
		}

		switch {
		case hub.IsJoin(inbound.Event):
			hub.AddClientToGroup(inbound.ClientID, hub.GroupID(inbound.Event))
		case hub.IsLeave(inbound.Event):
			hub.RemoveClientFromGroup(inbound.ClientID, hub.GroupID(inbound.Event))
		}
		return nil
	}
}

func setupMultiSink(
	cfg *app.Config,
	eventCodec codec.EventCodec[*kafkapb.KafkaEvent],
) (sink.Sink[*kafkapb.KafkaEvent], func() error, error) {
	kafkaSink, err := kafkaadapter.NewEventSink[*kafkapb.KafkaEvent](
		cfg.Kafka.Brokers,
		cfg.Kafka.InboundTopic,
		eventCodec,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("build kafka event sink: %w", err)
	}

	retryKafkaSink := sink.NewRetrySink[*kafkapb.KafkaEvent](kafkaSink, sink.RetrySinkConfig{Attempts: 3})
	asyncKafkaSink := sink.NewAsyncSink[*kafkapb.KafkaEvent](retryKafkaSink, sink.AsyncSinkConfig{
		BufferSize:     1024,
		Workers:        1,
		BlockOnEnqueue: true,
		OnWriteError: func(err error) {
			log.Printf("async kafka sink write error: %v", err)
		},
	})

	multiSink := sink.NewMultiSink[*kafkapb.KafkaEvent](sink.MultiSinkConfig{Concurrent: false}, asyncKafkaSink)
	return multiSink, asyncKafkaSink.Close, nil
}

func setupHandlerChain(
	hub *gateway.Hub[*kafkapb.KafkaEvent],
	multiSink sink.Sink[*kafkapb.KafkaEvent],
	jwtMiddleware handler.Middleware,
) handler.HandlerFunc {
	assignGroup := groupAssignmentHandler(hub)
	rateLimitMiddleware := middlewares.ConnectionRateLimitMiddleware(middlewares.ConnectionRateLimitOptions{
		RatePerSecond: 20,
		Burst:         40,
		IdleTTL:       5 * time.Minute,
	})

	groupAssignmentMiddleware := func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(c *handler.Context) error {
			if err := assignGroup(c); err != nil {
				return err
			}

			inbound, ok := c.Event.(gateway.InboundEvent[*kafkapb.KafkaEvent])
			if !ok {
				return fmt.Errorf("unexpected event type: %T", c.Event)
			}
			if !hub.IsMessage(inbound.Event) {
				return nil
			}
			return next(c)
		}
	}
	// TODO: add logging middleware, etc.
	// TODO: add jwt auth middleware after deciced jwt claims structure and how to pass jwt from client (e.g., via query param or subprotocol)
	finalSinkHandler := handler.SinkHandler(messageEventSinkWriter(hub, multiSink))
	return handler.NewHandlerChain(finalSinkHandler, rateLimitMiddleware, groupAssignmentMiddleware).Build()
}

func main() {
	cfg := mustLoadConfig("configs/config.yaml")

	eventCodec := newEventCodec(cfg.Event.Codec)
	hub := newHub(eventCodec)

	multiSink, closeSink := mustSetupMultiSink(cfg, eventCodec)
	defer closeSink()

	jwtMiddleware := newJWTMiddleware()
	inboundHandler := setupHandlerChain(hub, multiSink, jwtMiddleware)

	mux := newMux(hub, inboundHandler)
	fanoutSource := source.NewFanoutHTTPHandler(hub, cfg.Fanout.Address)
	if err := fanoutSource.Start(context.Background()); err != nil {
		log.Fatalf("failed to start fanout http source: %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := fanoutSource.Stop(stopCtx); err != nil {
			log.Printf("failed to stop fanout http source: %v", err)
		}
	}()

	if err := http.ListenAndServe(cfg.Server.Address, mux); err != nil {
		log.Fatal(err)
	}
}

func mustLoadConfig(path string) *app.Config {
	cfg, err := app.LoadConfig(path)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	return cfg
}

func newHub(eventCodec codec.EventCodec[*kafkapb.KafkaEvent]) *gateway.Hub[*kafkapb.KafkaEvent] {
	eventRouter := gateway.EventRouter[*kafkapb.KafkaEvent]{
		MsgType: func(e *kafkapb.KafkaEvent) string { return e.MsgType },
		GroupID: func(e *kafkapb.KafkaEvent) uint32 { return e.RoomId },
	}
	return gateway.NewHub(gateway.NewMemoryStore(), eventCodec, eventRouter)
}

func mustSetupMultiSink(
	cfg *app.Config,
	eventCodec codec.EventCodec[*kafkapb.KafkaEvent],
) (sink.Sink[*kafkapb.KafkaEvent], func()) {
	multiSink, closeSink, err := setupMultiSink(cfg, eventCodec)
	if err != nil {
		log.Fatalf("failed to setup multi sink: %v", err)
	}
	return multiSink, func() {
		if err := closeSink(); err != nil {
			log.Printf("failed to close inbound sink: %v", err)
		}
	}
}

func newJWTMiddleware() handler.Middleware {
	return middlewares.JWTAuthMiddleware[*ConnectionJWTClaims](middlewares.JWTAuthOptions[*ConnectionJWTClaims]{
		NewClaims: func() *ConnectionJWTClaims {
			return &ConnectionJWTClaims{}
		},
		Keyfunc: middlewares.KeyfuncByAlgorithm(map[string]any{
			jwt.SigningMethodHS256.Alg(): []byte(devSharedJWTSecret),
			jwt.SigningMethodHS384.Alg(): []byte(devSharedJWTSecret),
			jwt.SigningMethodHS512.Alg(): []byte(devSharedJWTSecret),
		}),
	})
}

func newMux(
	hub *gateway.Hub[*kafkapb.KafkaEvent],
	inboundHandler handler.HandlerFunc,
) *http.ServeMux {
	mux := http.NewServeMux()
	wsHandler := WsHandler(hub, inboundHandler)
	globalConnLimiter := middlewares.GlobalConnectionRateLimitMiddleware(middlewares.GlobalConnectionRateLimitOptions{
		RatePerSecond: 30,
		Burst:         60,
	})
	mux.Handle("/ws", globalConnLimiter(wsHandler))
	return mux
}

func newEventCodec(codecType string) codec.EventCodec[*kafkapb.KafkaEvent] {
	switch codecType {
	case "json":
		return codec.NewJSONEventCodec[*kafkapb.KafkaEvent]()
	default:
		pbCodec, err := codec.NewProtobufEventCodec(func() *kafkapb.KafkaEvent {
			return &kafkapb.KafkaEvent{}
		})
		if err != nil {
			log.Fatalf("failed to build protobuf codec: %v", err)
		}
		return pbCodec
	}
}
