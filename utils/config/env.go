package config

import (
	"loan-service/utils/auth"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

type Config struct {
	DBHost string `env:"DB_HOST"`
	DBPort string `env:"DB_PORT"`
	DBUser string `env:"DB_USER"`
	DBPass string `env:"DB_PASS"`
	DBName string `env:"DB_NAME"`

	RedisHost string `env:"REDIS_HOST"`
	RedisPort string `env:"REDIS_PORT"`

	AuthSecret string `env:"AUTH_SECRET"`
	Authorizer *auth.Authorizer
}

var Conf Config

func LoadConfig() error {
	_ = godotenv.Load()

	if err := env.Parse(&Conf); err != nil {
		return err
	}

	return nil
}
