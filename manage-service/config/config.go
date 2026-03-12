package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		App  `yaml:"app"`
		HTTP `yaml:"http"`
		Log  `yaml:"logger"`
		PG   `yaml:"postgres"`
		MinIO `yaml:"minio"`
		Extraction `yaml:"extraction"`
		RabbitMQ `yaml:"rabbitmq"`
	}

	App struct {
		Name    string `env-required:"true" yaml:"name"    env:"APP_NAME"`
		Version string `env-required:"true" yaml:"version" env:"APP_VERSION"`
	}

	HTTP struct {
		Port string `env-required:"true" yaml:"port" env:"HTTP_PORT"`
	}

	Log struct {
		Level string `env-required:"true" yaml:"log_level"   env:"LOG_LEVEL"`
	}

	PG struct {
		PoolMax int    `env-required:"true" yaml:"pool_max" env:"PG_POOL_MAX"`
		URL     string `env-required:"true" yaml:"pg_url"   env:"PG_URL"`
	}

	MinIO struct {
		Endpoint  string `env-required:"true" yaml:"endpoint" env:"MINIO_ENDPOINT"`
		AccessKey string `env-required:"true" yaml:"access_key" env:"MINIO_ACCESS_KEY"`
		SecretKey string `env-required:"true" yaml:"secret_key" env:"MINIO_SECRET_KEY"`
		Bucket    string `env-required:"true" yaml:"bucket" env:"MINIO_BUCKET"`
	}

	Extraction struct {
		ServiceURL string `env-required:"true" yaml:"service_url" env:"EXTRACTION_SERVICE_URL"`
	}

	RabbitMQ struct {
		URL string `env-required:"true" yaml:"url" env:"RABBITMQ_URL"`
	}
)

func NewConfig() (*Config, error) {
	cfg := &Config{}

	err := cleanenv.ReadConfig("./config/config.yml", cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	err = cleanenv.ReadEnv(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
