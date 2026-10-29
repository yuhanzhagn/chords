package kafka

import (
	"context"
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
	"time"
)

func TestLazyProducerUnavailableAndCanceledLifecycle(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	listener.Close()
	producer := NewKafkaProducer([]string{address})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	require.Error(t, producer.PublishContext(ctx, "notification", []byte("1"), []byte("test")))
	require.Less(t, time.Since(started), time.Second)
	ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	require.Error(t, producer.PublishContext(ctx, "notification", []byte("1"), []byte("test")))
	require.NoError(t, producer.Close())
	require.NoError(t, producer.Close())
	require.Error(t, producer.PublishContext(context.Background(), "notification", nil, nil))
}

func TestLazyProducerRecoversAndWaitsForKafkaProtocolACK(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	listener.Close()
	producer := NewKafkaProducer([]string{address})
	defer producer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.Error(t, producer.PublishContext(ctx, "notification", []byte("7"), []byte("immutable")))
	broker := sarama.NewMockBrokerAddr(t, 1, address)
	defer broker.Close()
	handlers := map[string]sarama.MockResponse{
		"ApiVersionsRequest": sarama.NewMockApiVersionsResponse(t),
		"MetadataRequest":    sarama.NewMockMetadataResponse(t).SetBroker(broker.Addr(), 1).SetLeader("notification", 0, 1),
		"ProduceRequest":     sarama.NewMockProduceResponse(t).SetError("notification", 0, sarama.ErrNotEnoughReplicas),
	}
	broker.SetHandlerByMap(handlers)
	require.Error(t, producer.PublishContext(ctx, "notification", []byte("7"), []byte("immutable")), "broker rejection must not count as publication")
	handlers["ProduceRequest"] = sarama.NewMockProduceResponse(t).SetError("notification", 0, sarama.ErrNoError)
	broker.SetHandlerByMap(handlers)
	require.NoError(t, producer.PublishContext(ctx, "notification", []byte("7"), []byte("immutable")))
	require.GreaterOrEqual(t, len(broker.History()), 3)
	require.NoError(t, producer.Close())
}
