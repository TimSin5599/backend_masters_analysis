package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		App   `yaml:"app"`
		HTTP  `yaml:"http"`
		Log   `yaml:"logger"`
		PG    `yaml:"postgres"`
		JWT   `yaml:"jwt"`
		Redis `yaml:"redis"`
		CORS  `yaml:"cors"`
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

	JWT struct {
		SignKey string `env-required:"true" yaml:"sign_key" env:"JWT_SIGN_KEY"`
	}

	Redis struct {
		URL string `env-required:"true" yaml:"url" env:"REDIS_URL"`
	}

	CORS struct {
		AllowOrigin string `yaml:"allow_origin" env:"CORS_ALLOW_ORIGIN"`
	}
)

func NewConfig() (*Config, error) {
	cfg := &Config{}

	if _, err := os.Stat("./config/config.yml"); err == nil {
		if err = cleanenv.ReadConfig("./config/config.yml", cfg); err != nil {
			return nil, fmt.Errorf("config error: %w", err)
		}
	}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
