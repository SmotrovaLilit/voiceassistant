package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"voiceassistant-api/internal/adapters/googlesheetscrm"
	"voiceassistant-api/internal/adapters/localai"
	"voiceassistant-api/internal/api/httpapi"
	"voiceassistant-api/internal/application"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	err := run(ctx, logger)
	stop()
	if err != nil {
		logger.ErrorContext(ctx, "API stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}
	client, err := localai.New(localai.Config{
		BaseURL: config.localAIURL, APIKey: os.Getenv("LOCALAI_API_KEY"),
		TranscriptionModel: config.transcriptionModel, AnalysisModel: config.analysisModel,
		Timezone: config.timezone, HTTPClient: &http.Client{Timeout: config.processingTimeout},
	})
	if err != nil {
		return fmt.Errorf("configure LocalAI: %w", err)
	}
	crm, err := googlesheetscrm.New(ctx, googlesheetscrm.Config{
		SpreadsheetID:   config.spreadsheetID,
		CredentialsJSON: config.credentialsJSON, CredentialsFile: config.credentialsFile,
	})
	if err != nil {
		return fmt.Errorf("configure Google Sheets CRM: %w", err)
	}
	service := application.NewService(crm, client, client)
	gin.SetMode(gin.ReleaseMode)
	router := httpapi.NewRouter(httpapi.Config{
		Service: service, MaxAudioBytes: config.maxAudioBytes,
		ProcessingTimeout: config.processingTimeout, Logger: logger,
	})
	server := &http.Server{
		Addr: config.address, Handler: router,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 2 * time.Minute,
		WriteTimeout: config.processingTimeout + time.Minute, IdleTimeout: time.Minute,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	var listenerConfig net.ListenConfig
	listener, err := listenerConfig.Listen(ctx, "tcp", config.address)
	if err != nil {
		return fmt.Errorf("listen for HTTP: %w", err)
	}
	logger.InfoContext(ctx, "API listening", "address", listener.Addr().String())
	stopped := make(chan error, 1)
	go func() { stopped <- server.Serve(listener) }()
	select {
	case err := <-stopped:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return errors.Join(err, server.Close())
		}
		return nil
	}
}
