// Code shared by all Go consumers. Edit proto/kafka/admission.go, then run make proto.
package kafkapb

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"unicode/utf8"

	"google.golang.org/protobuf/proto"
)

const (
	MaxKafkaRecordBytes      = 1 << 20 // Sarama default Producer.MaxMessageBytes
	MaxContentBytes          = 256 << 10
	MaxIdentifierBytes       = 128
	MaxTransportBytes        = 1 << 20
	MaxCommandHeaderBytes    = 4096
	KafkaRecordOverheadBytes = 1024                     // includes record framing, key and header names
	MaxRESTBodyBytes         = 6*MaxContentBytes + 4096 // escaped JSON content, identifiers, fixed envelope
)

var ErrMessageTooLarge = errors.New("message too large")
var ErrInvalidMessage = errors.New("invalid message content or identifier")

// ValidateInput applies the same UTF-8 and identifier policy at REST and WS ingress.
func ValidateInput(content []byte, requestID, tempID string) error {
	if len(content) > MaxContentBytes || len(requestID) > MaxIdentifierBytes || len(tempID) > MaxIdentifierBytes {
		return ErrMessageTooLarge
	}
	if !utf8.Valid(content) {
		return ErrInvalidMessage
	}
	for _, id := range []string{requestID, tempID} {
		for i := 0; i < len(id); i++ {
			if id[i] < 0x21 || id[i] > 0x7e {
				return ErrInvalidMessage
			}
		}
	}
	return nil
}

// ValidateSend reserves worst-case server metadata, so accepted input fits all transports.
// Unknown/client-only fields never enter the durable snapshot; incoming wire size is checked separately.
func ValidateSend(e *KafkaEvent) error {
	if e == nil {
		return ErrInvalidMessage
	}
	if err := ValidateInput(e.Content, e.RequestId, e.TempId); err != nil {
		return err
	}
	if proto.Size(e)+MaxCommandHeaderBytes+KafkaRecordOverheadBytes > MaxKafkaRecordBytes {
		return ErrMessageTooLarge
	}
	worst := &KafkaEvent{Id: math.MaxUint64, UserId: math.MaxUint32, RoomId: math.MaxUint32, MsgType: "message", Content: e.Content,
		TempId: e.TempId, RequestId: strings.Repeat("<", MaxIdentifierBytes), EventId: strings.Repeat("x", 32), ServerMessageId: math.MaxUint64,
		RoomSeq: math.MaxInt64, CreatedAt: math.MaxInt64, PersistedAt: "9999-12-31T23:59:59.999999999Z"}
	if proto.Size(worst)+MaxCommandHeaderBytes+KafkaRecordOverheadBytes > MaxKafkaRecordBytes {
		return ErrMessageTooLarge
	}
	body, err := json.Marshal(struct {
		RoomID uint32      `json:"room_id"`
		Event  *KafkaEvent `json:"event"`
	}{math.MaxUint32, worst})
	if err != nil {
		return err
	}
	if len(body) > MaxTransportBytes {
		return ErrMessageTooLarge
	}
	return nil
}
