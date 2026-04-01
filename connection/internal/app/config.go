package app

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Address string `yaml:"address"`
	} `yaml:"server"`
	Fanout struct {
		Address       string `yaml:"address"`
		AdvertiseAddr string `yaml:"advertise_addr"`
	} `yaml:"fanout"`
	Event struct {
		Codec string `yaml:"codec"`
	} `yaml:"event"`
	Kafka struct {
		Brokers      []string `yaml:"brokers"`
		InboundTopic string   `yaml:"inbound_topic"`
	} `yaml:"kafka"`
	Redis struct {
		Addr              string        `yaml:"addr"`
		Password          string        `yaml:"password"`
		DB                int           `yaml:"db"`
		RoomUsersPrefix   string        `yaml:"room_users_prefix"`
		RoomUsersSuffix   string        `yaml:"room_users_suffix"`
		UserGatewayPrefix string        `yaml:"user_gateway_prefix"`
		UserGatewaySuffix string        `yaml:"user_gateway_suffix"`
		RoomUsersTTL      time.Duration `yaml:"room_users_ttl"`
		UserGatewayTTL    time.Duration `yaml:"user_gateway_ttl"`
		PresenceRefresh   time.Duration `yaml:"presence_refresh_interval"`
	} `yaml:"redis"`
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
	if c.Fanout.Address == "" {
		c.Fanout.Address = ":8082"
	}
	if c.Fanout.AdvertiseAddr == "" {
		c.Fanout.AdvertiseAddr = "connection:8082"
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
	if c.Redis.RoomUsersTTL == 0 {
		c.Redis.RoomUsersTTL = 2 * time.Minute
	}
	if c.Redis.UserGatewayTTL == 0 {
		c.Redis.UserGatewayTTL = 2 * time.Minute
	}
	if c.Redis.PresenceRefresh == 0 {
		c.Redis.PresenceRefresh = 30 * time.Second
	}
}
