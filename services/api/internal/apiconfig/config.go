package apiconfig

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// LoadConfig reads configs/config.yaml from the working directory, then the
// Postgres password from the file the config names.
// It returns an error when either file is missing or does not parse, or the config is not valid.
func LoadConfig(logger *zap.Logger) (*Config, error) {
	var cfg Config

	v := viper.New()
	v.SetConfigType("yaml")
	v.AddConfigPath("configs")
	v.SetConfigName("config")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	password, err := os.ReadFile(cfg.Postgres.PasswordFile)
	if err != nil {
		return nil, fmt.Errorf("read postgres password: %w", err)
	}
	// Secret files often end with a newline that is not part of the password.
	cfg.Postgres.Password = strings.TrimSpace(string(password))
	if cfg.Postgres.Password == "" {
		return nil, fmt.Errorf("postgres password file %s is empty", cfg.Postgres.PasswordFile)
	}

	logger.Info("Successfully read config", zap.String("file", v.ConfigFileUsed()), zap.Any("config", cfg))
	return &cfg, nil
}
