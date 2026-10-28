package kafka

import (
	"connection/internal/event/codec"
	kafkapb "connection/proto/kafka"
	"context"
	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCommandAuthorizationHeader(t *testing.T) {
	producer := mocks.NewSyncProducer(t, nil)
	producer.ExpectSendMessageWithMessageCheckerFunctionAndSucceed(func(message *sarama.ProducerMessage) error {
		require.Equal(t, "user-request", message.Topic)
		require.Equal(t, []sarama.RecordHeader{{Key: []byte("chords-command"), Value: []byte("short-grant")}}, message.Headers)
		return nil
	})
	c, err := codec.NewProtobufEventCodec(func() *kafkapb.KafkaEvent { return &kafkapb.KafkaEvent{} })
	require.NoError(t, err)
	s := &EventSink[*kafkapb.KafkaEvent]{producer: producer, topic: "user-request", codec: c}
	require.NoError(t, s.Write(WithCommandToken(context.Background(), "short-grant"), &kafkapb.KafkaEvent{MsgType: "message", UserId: 42, RoomId: 7}))
	require.NoError(t, s.Close())
}
