package config

import (
	"time"
)

type Config struct {
	GRPC struct {
		Port            int           `yaml:"port"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	} `yaml:"grpc"`
	Log struct {
		Level string `yaml:"level"`
	} `yaml:"log"`
}

func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.GRPC.Port = 50051
	cfg.GRPC.ShutdownTimeout = 10 * time.Second
	cfg.Log.Level = "info"
	return cfg
}
