package app

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Kafka struct {
		Brokers       []string `yaml:"brokers"`
		ConsumerGroup string   `yaml:"consumer_group"`
		Topics        []string `yaml:"topics"`
	} `yaml:"kafka"`
	Redis struct {
		Addr              string `yaml:"addr"`
		Password          string `yaml:"password"`
		DB                int    `yaml:"db"`
		RoomUsersPrefix   string `yaml:"room_users_prefix"`
		RoomUsersSuffix   string `yaml:"room_users_suffix"`
		UserGatewayPrefix string `yaml:"user_gateway_prefix"`
		UserGatewaySuffix string `yaml:"user_gateway_suffix"`
	} `yaml:"redis"`
	Fanout struct {
		GatewayPath    string        `yaml:"gateway_path"`
		RequestTimeout time.Duration `yaml:"request_timeout"`
	} `yaml:"fanout"`
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
	fmt.Printf("Fanout config: %+v\n", cfg)
	return &cfg, nil
}

func (c *Config) withDefaults() {
	if len(c.Kafka.Brokers) == 0 {
		c.Kafka.Brokers = []string{"kafka:9092"}
	}
	if c.Kafka.ConsumerGroup == "" {
		c.Kafka.ConsumerGroup = "fanout-workers"
	}
	if len(c.Kafka.Topics) == 0 {
		c.Kafka.Topics = []string{"notification"}
	}
	if c.Redis.Addr == "" {
		c.Redis.Addr = "redis:6379"
	}
	if c.Redis.RoomUsersPrefix == "" {
		c.Redis.RoomUsersPrefix = "room:"
	}
	if c.Redis.RoomUsersSuffix == "" {
		c.Redis.RoomUsersSuffix = ":users"
	}
	if c.Redis.UserGatewayPrefix == "" {
		c.Redis.UserGatewayPrefix = "user:"
	}
	if c.Redis.UserGatewaySuffix == "" {
		c.Redis.UserGatewaySuffix = ":gateway"
	}
	if c.Fanout.GatewayPath == "" {
		c.Fanout.GatewayPath = "/fanout"
	}
	if c.Fanout.RequestTimeout == 0 {
		c.Fanout.RequestTimeout = 3 * time.Second
	}
}
