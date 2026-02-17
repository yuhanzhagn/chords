package main

import (
	"connection/internal/platform/codec"
	"connection/internal/platform/kafka"
	"connection/internal/service"
	"connection/internal/transport/websocket"
	kafkapb "connection/proto/kafka"
	"context"
	"log"
	"net/http"
	"os"
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

func StartKafkaConsumer[T any](hub *websocket.Hub[T]) {
	consumer, err := kafka.NewWsOutboundConsumer(
		[]string{"kafka:9092"},
		"connection-ws-gateway",
		[]string{"notification"},
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
	// HTTP server setup
	mux := http.NewServeMux()
	eventCodec := newEventCodec()
	eventRouter := websocket.EventRouter[*kafkapb.KafkaEvent]{
		MsgType: func(e *kafkapb.KafkaEvent) string { return e.MsgType },
		RoomID:  func(e *kafkapb.KafkaEvent) uint32 { return e.RoomId },
	}
	hub := websocket.NewHub(websocket.NewMemoryStore(), eventCodec, eventRouter)

	msgService := &service.MessageService{
		Producer: kafka.NewKafkaProducer([]string{"kafka:9092"}),
	}

	wsHandler := WsAuthMiddleware(WsHandler(hub, msgService))
	mux.Handle("/ws", wsHandler)

	// Start Kafka consumer
	StartKafkaConsumer(hub)

	http.ListenAndServe(":8081", mux)
}

func newEventCodec() codec.EventCodec[*kafkapb.KafkaEvent] {
	switch os.Getenv("EVENT_CODEC") {
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
