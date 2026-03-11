package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	App struct {
		Port                int    `env:"APP_PORT"                  env-default:"8080"`
		Env                 string `env:"APP_ENV"                   env-default:"development"`
		ReadTimeoutSeconds  int    `env:"APP_READ_TIMEOUT_SECONDS"  env-default:"15"`
		WriteTimeoutSeconds int    `env:"APP_WRITE_TIMEOUT_SECONDS" env-default:"15"`
		IdleTimeoutSeconds  int    `env:"APP_IDLE_TIMEOUT_SECONDS"  env-default:"60"`
	}
	MongoDB struct {
		URI            string `env:"MONGODB_URI"              env-required:"true"`
		Database       string `env:"MONGODB_DATABASE"         env-required:"true"`
		TimeoutSeconds int    `env:"MONGODB_TIMEOUT_SECONDS"  env-default:"10"`
	}
	RabbitMQ struct {
		URI        string `env:"RABBITMQ_URI"         env-required:"true"`
		Exchange   string `env:"RABBITMQ_EXCHANGE"    env-default:"orders"`
		Queue      string `env:"RABBITMQ_QUEUE"       env-required:"true"`
		RoutingKey string `env:"RABBITMQ_ROUTING_KEY" env-required:"true"`
	}
	Logger struct {
		Level string `env:"LOGGER_LEVEL" env-default:"info"`
	}
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadConfig(".env", cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
