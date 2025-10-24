package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Host             string
	DatabasePostgres string
}

func NewConfig() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.Host, "host", "", "HTTP-server address (host:port)")
	flag.StringVar(&cfg.DatabasePostgres, "database-postgres", "", "PostgreSQL connection string")

	flag.Parse()

	cfg.loadFromEnv()
	cfg.setDefaults()

	if err := cfg.validate(); err != nil {
		panic(fmt.Sprintf("Invalid configuration: %v", err))
	}

	return cfg
}

func (c *Config) loadFromEnv() {
	if envHost := os.Getenv("HOST"); envHost != "" {
		c.Host = envHost
	}

	if envDatabasePostgres := os.Getenv("POSTGRES_URL"); envDatabasePostgres != "" {
		c.DatabasePostgres = envDatabasePostgres
	}
}

func (c *Config) setDefaults() {
	if c.Host == "" {
		c.Host = "localhost:8080"
	}
	if c.DatabasePostgres == "" {
		c.DatabasePostgres = "postgres://postgres:postgres@localhost:5432/crud?sslmode=disable"
	}
}

func (c *Config) validate() error {
	if c.Host == "" {
		return fmt.Errorf("хост должен быть указан")
	}

	if !strings.Contains(c.Host, ":") {
		return fmt.Errorf("хост должен иметь такую структуру (host:port)")
	}

	if c.DatabasePostgres == "" {
		return fmt.Errorf("строка подключения базы данных обязательна")
	}

	if !strings.Contains(c.DatabasePostgres, "postgres://") {
		return fmt.Errorf("не валидная строка подключения к базе данных")
	}

	return nil
}
