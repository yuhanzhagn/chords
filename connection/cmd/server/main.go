package main

import (
	"connection/internal/app"
	"connection/internal/platform/codec"
	"connection/internal/platform/kafka"
	"connection/internal/service"
	"connection/internal/transport/websocket"
	kafkapb "connection/proto/kafka"
	"context"
	"log"
	"net/http"
)

func WsAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := 10, true // mock authentication
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// put userID in context
		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WsHandler[T any](hub *websocket.Hub[T], msgService *service.MessageService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// userID := r.Context().Value("userID").(uint32)
		userID := uint32(10)
		websocket.ServeWs(userID, w, r, hub, msgService)
	})
}

func StartKafkaConsumer[T any](hub *websocket.Hub[T], cfg *app.Config) {
	consumer, err := kafka.NewWsOutboundConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.ConsumerGroup,
		cfg.Kafka.OutboundTopics,
		hub.HandleOutboundEvent,
		hub.Codec(),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx, _ := context.WithCancel(context.Background())
	// defer cancel()

	consumer.Start(ctx)
}

func main() {
	cfg, err := app.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// HTTP server setup
	mux := http.NewServeMux()
	eventCodec := newEventCodec(cfg.Event.Codec)
	eventRouter := websocket.EventRouter[*kafkapb.KafkaEvent]{
		MsgType: func(e *kafkapb.KafkaEvent) string { return e.MsgType },
		RoomID:  func(e *kafkapb.KafkaEvent) uint32 { return e.RoomId },
	}
	hub := websocket.NewHub(websocket.NewMemoryStore(), eventCodec, eventRouter)

	msgService := &service.MessageService{
		Producer:     kafka.NewKafkaProducer(cfg.Kafka.Brokers),
		InboundTopic: cfg.Kafka.InboundTopic,
	}

	wsHandler := WsAuthMiddleware(WsHandler(hub, msgService))
	mux.Handle("/ws", wsHandler)

	// Start Kafka consumer
	StartKafkaConsumer(hub, cfg)

	if err := http.ListenAndServe(cfg.Server.Address, mux); err != nil {
		log.Fatal(err)
	}
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
