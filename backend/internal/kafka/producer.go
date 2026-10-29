package kafka

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

// LazyProducer permits HTTP persistence while the notification broker is down.
// One relay owns it; Sarama connections and acknowledgment waits are bounded.
type KafkaProducer struct {
	brokers  []string
	mu       sync.Mutex
	producer sarama.AsyncProducer
	done     chan struct{}
	closed   bool
}

func NewKafkaProducer(brokers []string) *KafkaProducer { return &KafkaProducer{brokers: brokers} }
func brokerConfig() *sarama.Config {
	c := sarama.NewConfig()
	c.Version = sarama.V2_8_0_0
	c.Net.DialTimeout = time.Second
	c.Net.ReadTimeout = 2 * time.Second
	c.Net.WriteTimeout = 2 * time.Second
	c.Metadata.Retry.Max = 0
	c.Metadata.Timeout = 2 * time.Second
	c.Producer.RequiredAcks = sarama.WaitForAll
	c.Producer.Return.Successes = true
	c.Producer.Return.Errors = true
	c.Producer.Timeout = 2 * time.Second
	c.Producer.Retry.Max = 0
	return c
}
func (p *KafkaProducer) connect(ctx context.Context) (sarama.AsyncProducer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, errors.New("producer closed")
	}
	if p.producer != nil {
		return p.producer, nil
	}
	type outcome struct {
		p   sarama.AsyncProducer
		err error
	}
	ch := make(chan outcome)
	go func() {
		v, e := sarama.NewAsyncProducer(p.brokers, brokerConfig())
		select {
		case ch <- outcome{v, e}:
		case <-ctx.Done():
			if v != nil {
				v.AsyncClose()
				go drain(v, nil)
			}
		}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-ch:
		if result.err != nil {
			return nil, result.err
		}
		p.producer = result.p
		p.done = make(chan struct{})
		go drain(result.p, p.done)
		return result.p, nil
	}
}
func drain(p sarama.AsyncProducer, done chan struct{}) {
	if done != nil {
		defer close(done)
	}
	successes, failures := p.Successes(), p.Errors()
	for successes != nil || failures != nil {
		select {
		case msg, ok := <-successes:
			if !ok {
				successes = nil
				continue
			}
			if ack, ok := msg.Metadata.(chan error); ok {
				ack <- nil
			}
		case failure, ok := <-failures:
			if !ok {
				failures = nil
				continue
			}
			if ack, ok := failure.Msg.Metadata.(chan error); ok {
				ack <- failure.Err
			}
		}
	}
}
func (p *KafkaProducer) PublishContext(ctx context.Context, topic string, key, value []byte) error {
	producer, err := p.connect(ctx)
	if err != nil {
		return err
	}
	ack := make(chan error, 1)
	msg := &sarama.ProducerMessage{Topic: topic, Key: sarama.ByteEncoder(key), Value: sarama.ByteEncoder(value), Metadata: ack}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case producer.Input() <- msg:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ack:
		return err
	}
}
func (p *KafkaProducer) Publish(topic string, key, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.PublishContext(ctx, topic, key, value)
}
func (p *KafkaProducer) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	producer, done := p.producer, p.done
	p.mu.Unlock()
	if producer == nil {
		return nil
	}
	producer.AsyncClose()
	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("producer close timeout")
	}
}
