package apiconfig

import (
	"fmt"
	"time"
)

type Config struct {
	API      APIConfig      `mapstructure:"api"`
	Postgres PostgresConfig `mapstructure:"postgres"`
}

type APIConfig struct {
	Addr            string        `mapstructure:"addr"`
	Port            int           `mapstructure:"port"`
	HealthCheckPort int           `mapstructure:"healthCheckPort"`
	LogLevel        string        `mapstructure:"logLevel"`
	ShutdownTimeout time.Duration `mapstructure:"shutdownTimeout"`

	// Version is set from the build, not from the file.
	Version string `mapstructure:"-"`
}

// PostgresConfig holds no password. The password comes from PasswordFile, so
// the config can live in a ConfigMap and the password in a mounted Secret.
type PostgresConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Database     string `mapstructure:"database"`
	SSLMode      string `mapstructure:"sslMode"`
	PasswordFile string `mapstructure:"passwordFile"`

	// Password is read from PasswordFile, not from the config.
	Password string `mapstructure:"-" json:"-"`
}

func (c Config) Validate() error {
	if err := c.API.Validate(); err != nil {
		return fmt.Errorf("api: %w", err)
	}
	if err := c.Postgres.Validate(); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	return nil
}

func (c APIConfig) Validate() error {
	if c.Port <= 0 {
		return fmt.Errorf("port must be greater than 0")
	}
	if c.HealthCheckPort <= 0 {
		return fmt.Errorf("healthCheckPort must be greater than 0")
	}
	if c.HealthCheckPort == c.Port {
		return fmt.Errorf("healthCheckPort must differ from port")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdownTimeout must be greater than 0")
	}
	return nil
}

func (c PostgresConfig) Validate() error {
	if c.Host == "" || c.User == "" || c.Database == "" {
		return fmt.Errorf("host, user and database must be set")
	}
	if c.Port <= 0 {
		return fmt.Errorf("port must be greater than 0")
	}
	if c.PasswordFile == "" {
		return fmt.Errorf("passwordFile must be set")
	}
	return nil
}
