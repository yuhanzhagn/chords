package main

import (
	kafkaadapter "connection/internal/adapter/kafka"
	"connection/internal/app"
	"connection/internal/event/codec"
	"connection/internal/gateway"
	"connection/internal/handler"
	"connection/internal/sink"
	"connection/internal/source"
	kafkapb "connection/proto/kafka"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
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

func WsHandler[T any](hub *gateway.Hub[T], inboundHandler handler.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// userID := r.Context().Value("userID").(uint32)
		userID := uint32(10)
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
) handler.HandlerFunc {
	assignGroup := groupAssignmentHandler(hub)

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
	finalSinkHandler := handler.SinkHandler(messageEventSinkWriter(hub, multiSink))
	return handler.NewHandlerChain(finalSinkHandler, groupAssignmentMiddleware).Build()
}

func outboundHubEventWriter(hub *gateway.Hub[*kafkapb.KafkaEvent]) handler.SinkFunc {
	return func(_ context.Context, event any) error {
		msg, ok := event.(*kafkapb.KafkaEvent)
		if !ok {
			return fmt.Errorf("unexpected outbound event type: %T", event)
		}
		hub.HandleOutboundEvent(msg)
		return nil
	}
}

func setupOutboundHandlerChain(hub *gateway.Hub[*kafkapb.KafkaEvent]) handler.HandlerFunc {
	finalHubHandler := handler.SinkHandler(outboundHubEventWriter(hub))
	return handler.NewHandlerChain(finalHubHandler).Build()
}

func main() {
	cfg, err := app.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	mux := http.NewServeMux()
	eventCodec := newEventCodec(cfg.Event.Codec)
	eventRouter := gateway.EventRouter[*kafkapb.KafkaEvent]{
		MsgType: func(e *kafkapb.KafkaEvent) string { return e.MsgType },
		GroupID: func(e *kafkapb.KafkaEvent) uint32 { return e.RoomId },
	}
	hub := gateway.NewHub(gateway.NewMemoryStore(), eventCodec, eventRouter)

	multiSink, closeSink, err := setupMultiSink(cfg, eventCodec)
	if err != nil {
		log.Fatalf("failed to setup multi sink: %v", err)
	}
	defer func() {
		if err := closeSink(); err != nil {
			log.Printf("failed to close inbound sink: %v", err)
		}
	}()

	inboundHandler := setupHandlerChain(hub, multiSink)

	wsHandler := WsAuthMiddleware(WsHandler(hub, inboundHandler))
	mux.Handle("/ws", wsHandler)

	outboundHandler := setupOutboundHandlerChain(hub)
	outboundSource, err := source.NewKafkaSource[*kafkapb.KafkaEvent](outboundHandler, source.KafkaSourceOptions[*kafkapb.KafkaEvent]{
		Brokers: cfg.Kafka.Brokers,
		GroupID: cfg.Kafka.ConsumerGroup,
		Topics:  cfg.Kafka.OutboundTopics,
		Decoder: eventCodec,
		OnHandleError: func(err error) {
			log.Printf("kafka outbound source error: %v", err)
		},
	})
	if err != nil {
		log.Fatalf("failed to setup kafka outbound source: %v", err)
	}

	sourceCtx, cancelSource := context.WithCancel(context.Background())
	defer cancelSource()

	if err := outboundSource.Start(sourceCtx); err != nil {
		log.Fatalf("failed to start kafka outbound source: %v", err)
	}

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := outboundSource.Stop(stopCtx); err != nil {
			log.Printf("failed to stop outbound source: %v", err)
		}
	}()

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
