package app

import (
    "os"
	"fmt"
    "gopkg.in/yaml.v3"
)

type Config struct {
	App struct {
		Port int `yaml:"port"`
	}`yaml:"app"`
	FrontEnd struct {
		Port int `yaml:"port"`
	}`yaml:"frontend"`
    Database struct {
        Dialect string `yaml:"dialect"`
        DSN     string `yaml:"dsn"`
    } `yaml:"database"`
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

	fmt.Printf("Config:  %+v\n", cfg)

    return &cfg, nil
}

