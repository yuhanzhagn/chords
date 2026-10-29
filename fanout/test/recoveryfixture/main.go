// Integration helper: substitutes only the Kafka transport and Redis registry;
// dispatch, service authentication and HTTP delivery use the real worker code.
package main

import (
	"context"
	"encoding/json"
	"fanout/internal/fanout"
	kafkapb "fanout/proto/kafka"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

type registry struct{ gateway string }

func (r registry) RoomUsers(context.Context, uint32) ([]uint32, error) { return []uint32{1}, nil }
func (r registry) UserGateways(context.Context, []uint32) (map[uint32]string, error) {
	return map[uint32]string{1: r.gateway}, nil
}
func main() {
	dispatcher, err := fanout.NewDispatcher(registry{os.Getenv("CHORDS_PROBE_GATEWAY")}, &http.Client{Timeout: time.Second}, fanout.Config{Token: os.Getenv("FANOUT_TOKEN")})
	if err != nil {
		panic(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	server := &http.Server{ReadHeaderTimeout: time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(io.LimitReader(r.Body, kafkapb.MaxTransportBytes+1))
		if err != nil || len(raw) > kafkapb.MaxTransportBytes {
			w.WriteHeader(413)
			return
		}
		var event kafkapb.KafkaEvent
		if proto.Unmarshal(raw, &event) != nil {
			w.WriteHeader(400)
			return
		}
		if dispatcher.Dispatch(r.Context(), &event) != nil {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(204)
	})}
	go server.Serve(listener)
	_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"url": "http://" + listener.Addr().String()})
	_, _ = io.Copy(io.Discard, os.Stdin)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
