package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/trybyteful/interview-takehome/services/api/internal/apiconfig"
	"github.com/trybyteful/interview-takehome/services/api/internal/logger"
	"github.com/trybyteful/interview-takehome/services/api/internal/server"
	"go.uber.org/zap"
)

var version string

func main() {

	basicLogger, lvl, err := logger.Basic()
	if err != nil {
		log.Fatal("Could not create logger")
	}
	defer basicLogger.Sync()

	cfg, err := apiconfig.LoadConfig(basicLogger)
	if err != nil {
		basicLogger.Fatal("Could not load config", zap.Error(err))
	}

	cfg.API.Version = version

	appLogger := logger.ConfigureLogger(basicLogger, cfg.API)

	if err := lvl.UnmarshalText([]byte(cfg.API.LogLevel)); err != nil {
		appLogger.Fatal("Invalid log level", zap.String("log_level", cfg.API.LogLevel))
	}

	api, err := server.NewAPI(appLogger, cfg)
	if err != nil {
		appLogger.Fatal("Could not setup the API", zap.Error(err))
	}

	stop, cancelStop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelStop()

	go api.Run(stop)

	<-stop.Done()
	appLogger.Info("Shutting down gracefully ...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.API.ShutdownTimeout)
	defer cancel()

	api.Shutdown(ctx)

	appLogger.Info("Server shutdown complete")
}
