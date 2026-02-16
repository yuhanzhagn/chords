package main

import (
	"connection/internal/platform/kafka"
	"connection/internal/service"
	"connection/internal/transport/websocket"
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

func WsHandler(hub *websocket.Hub, msgService *service.MessageService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// userID := r.Context().Value("userID").(uint32)
		userID := uint32(10)
		websocket.ServeWs(userID, w, r, hub, msgService)
	})
}

func StartKafkaConsumer(hub *websocket.Hub) {
	consumer, err := kafka.NewWsOutboundConsumer(
		[]string{"kafka:9092"},
		"connection-ws-gateway",
		[]string{"notification"},
		hub.HandleOutboundEvent,
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
	hub := websocket.NewHub(websocket.NewMemoryStore())

	msgService := &service.MessageService{
		Producer: kafka.NewKafkaProducer([]string{"kafka:9092"}),
	}

	wsHandler := WsAuthMiddleware(WsHandler(hub, msgService))
	mux.Handle("/ws", wsHandler)

	// Start Kafka consumer
	StartKafkaConsumer(hub)

	http.ListenAndServe(":8081", mux)
}
