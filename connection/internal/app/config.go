package app

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Address string `yaml:"address"`
	} `yaml:"server"`
	Event struct {
		Codec string `yaml:"codec"`
	} `yaml:"event"`
	Kafka struct {
		Brokers      []string `yaml:"brokers"`
		InboundTopic string   `yaml:"inbound_topic"`
	} `yaml:"kafka"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	cfg.withDefaults()

	fmt.Printf("Connection config: %+v\n", cfg)
	return &cfg, nil
}

func (c *Config) withDefaults() {
	if c.Server.Address == "" {
		c.Server.Address = ":8081"
	}
	if c.Event.Codec == "" {
		c.Event.Codec = "protobuf"
	}
	if len(c.Kafka.Brokers) == 0 {
		c.Kafka.Brokers = []string{"kafka:9092"}
	}
	if c.Kafka.InboundTopic == "" {
		c.Kafka.InboundTopic = "user-request"
	}
}
