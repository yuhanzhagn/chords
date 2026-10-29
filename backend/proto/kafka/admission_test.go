package kafkapb

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestSendAdmissionTransportBudgets(t *testing.T) {
	for _, n := range []int{MaxContentBytes - 2, MaxContentBytes - 1, MaxContentBytes} {
		// Control chars/quotes stress JSON escaping; bytes must still be base64, not raw JSON text.
		content := bytes.Repeat([]byte{'\x00', '"', '\\'}, (n+2)/3)[:n]
		e := &KafkaEvent{Id: math.MaxUint64, UserId: math.MaxUint32, RoomId: math.MaxUint32, MsgType: "message", Content: content, TempId: strings.Repeat("<", 128), RequestId: strings.Repeat("<", 128), EventId: strings.Repeat("x", 32), ServerMessageId: math.MaxUint64, RoomSeq: math.MaxInt64, CreatedAt: math.MaxInt64, PersistedAt: "9999-12-31T23:59:59.999999999Z"}
		require.NoError(t, ValidateSend(e))
		raw, err := proto.Marshal(e)
		require.NoError(t, err)
		record := &sarama.ProducerMessage{Topic: "user-request", Key: sarama.StringEncoder("4294967295"), Value: sarama.ByteEncoder(raw), Headers: []sarama.RecordHeader{{Key: []byte("chords-command"), Value: bytes.Repeat([]byte("x"), MaxCommandHeaderBytes)}}}
		size := record.ByteSize(2)
		require.Equal(t, sarama.NewConfig().Producer.MaxMessageBytes, MaxKafkaRecordBytes)
		require.LessOrEqual(t, size, MaxKafkaRecordBytes)
		require.LessOrEqual(t, size-len(raw)-MaxCommandHeaderBytes, KafkaRecordOverheadBytes)
		envelope, err := json.Marshal(struct {
			RoomID uint32      `json:"room_id"`
			Event  *KafkaEvent `json:"event"`
		}{e.RoomId, e})
		require.NoError(t, err)
		require.Contains(t, string(envelope), base64.StdEncoding.EncodeToString(content))
		require.LessOrEqual(t, len(envelope), MaxTransportBytes)
		t.Logf("content=%d protobuf=%d Kafka record=%d HTTP envelope=%d", n, len(raw), size, len(envelope))
	}
	require.NoError(t, ValidateInput([]byte(strings.Repeat("界", MaxContentBytes/3)), "id", "id"))
	require.ErrorIs(t, ValidateInput([]byte(strings.Repeat("界", MaxContentBytes/3+1)), "id", "id"), ErrMessageTooLarge)
	require.ErrorIs(t, ValidateInput(bytes.Repeat([]byte("x"), MaxContentBytes+1), "id", "id"), ErrMessageTooLarge)
	for _, id := range []string{strings.Repeat("x", 129), strings.Repeat("x", 128) + "\""} {
		require.ErrorIs(t, ValidateInput(nil, id, ""), ErrMessageTooLarge)
	}
	for _, id := range []string{"中文", "with space", "with\nnewline"} {
		require.ErrorIs(t, ValidateInput(nil, id, ""), ErrInvalidMessage)
	}
	require.ErrorIs(t, ValidateInput([]byte{0xff}, "", ""), ErrInvalidMessage)
}

func TestAdditiveWireCompatibility(t *testing.T) {
	// Old wire has fields 1..7; unset new IDs retain legacy meanings and bytes round-trip.
	old := &KafkaEvent{Id: 9007199254740993, UserId: 42, RoomId: 7, MsgType: "message", Content: []byte("hello"), TempId: "old-client", CreatedAt: 1000}
	raw, err := proto.Marshal(old)
	require.NoError(t, err)
	var decoded KafkaEvent
	require.NoError(t, proto.Unmarshal(raw, &decoded))
	require.Empty(t, decoded.RequestId)
	require.Zero(t, decoded.ServerMessageId)
	require.Equal(t, old.Id, decoded.Id)
	decoded.RequestId = "new"
	decoded.EventId = "stable"
	decoded.ServerMessageId = math.MaxUint64
	decoded.RoomSeq = math.MaxInt64
	raw, err = proto.Marshal(&decoded)
	require.NoError(t, err)
	var reread KafkaEvent
	require.NoError(t, proto.Unmarshal(raw, &reread))
	require.Equal(t, uint64(math.MaxUint64), reread.ServerMessageId)
	require.Equal(t, old.Id, reread.Id)
	require.EqualValues(t, math.MaxInt64, reread.RoomSeq)
}
