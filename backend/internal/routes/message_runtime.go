package routes

import (
	"backend/internal/kafka"
	"backend/internal/repo"
	"backend/internal/service"
	"context"
	"gorm.io/gorm"
	"log"
	"os"
	"strings"
	"time"
)

// StartMessageRuntime owns the relay and optional legacy command consumer. Router
// construction never dials Kafka. Stop before closing the database.
func StartMessageRuntime(parent context.Context, db *gorm.DB) func() {
	ctx, cancel := context.WithCancel(parent)
	brokers := []string{"kafka:9092"}
	if raw := os.Getenv("CHORDS_KAFKA_BROKERS"); raw != "" {
		brokers = strings.Split(raw, ",")
	}
	producer := kafka.NewKafkaProducer(brokers)
	relay := service.NewRelay(repo.NewRepoContainer(db), producer)
	done := make(chan struct{})
	go func() { defer close(done); relay.Run(ctx) }()
	legacyDone := make(chan struct{})
	go func() {
		defer close(legacyDone)
		if os.Getenv("CHORDS_LEGACY_COMMANDS") != "1" {
			return
		}
		ks := service.KafkaService{MessageService: service.NewMessageService(repo.NewRepoContainer(db))}
		for ctx.Err() == nil {
			consumer, err := kafka.NewWsOutboundConsumer(brokers, "backend-ws-processor", []string{"user-request"}, ks.HandleOutboundEvent)
			if err == nil {
				consumer.Start(ctx)
				<-ctx.Done()
				_ = consumer.Close()
				return
			}
			log.Println("legacy command consumer unavailable")
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	}()
	return func() {
		cancel()
		<-done
		_ = producer.Close()
		select {
		case <-legacyDone:
		case <-time.After(5 * time.Second):
			log.Println("legacy consumer shutdown timed out")
		}
	}
}
