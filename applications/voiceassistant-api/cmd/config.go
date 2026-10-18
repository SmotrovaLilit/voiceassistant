package main

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type config struct {
	address            string
	spreadsheetID      string
	credentialsJSON    string
	credentialsFile    string
	localAIURL         string
	transcriptionModel string
	analysisModel      string
	maxAudioBytes      int64
	processingTimeout  time.Duration
	timezone           *time.Location
}

func loadConfig() (config, error) {
	maxAudioBytes, err := strconv.ParseInt(envOr("MAX_AUDIO_BYTES", "20971520"), 10, 64)
	if err != nil || maxAudioBytes <= 0 || maxAudioBytes > math.MaxInt64-(1<<20) {
		return config{}, errors.New("MAX_AUDIO_BYTES must be a positive byte limit that leaves room for multipart overhead")
	}
	timeout, err := time.ParseDuration(envOr("PROCESSING_TIMEOUT", "10m"))
	if err != nil || timeout <= 0 || timeout > time.Duration(math.MaxInt64)-time.Minute {
		return config{}, errors.New("PROCESSING_TIMEOUT must be a positive duration that leaves room for the HTTP write timeout")
	}
	location, err := time.LoadLocation(envOr("TIMEZONE", "Europe/Berlin"))
	if err != nil {
		return config{}, fmt.Errorf("load TIMEZONE: %w", err)
	}
	return config{
		address:            net.JoinHostPort(envOr("LISTEN_ADDRESS", "127.0.0.1"), envOr("PORT", "8081")),
		spreadsheetID:      envOr("GOOGLE_SPREADSHEET_ID", ""),
		credentialsJSON:    envOr("GOOGLE_CREDENTIALS_JSON", ""),
		credentialsFile:    envOr("GOOGLE_CREDENTIALS_FILE", ""),
		localAIURL:         envOr("LOCALAI_BASE_URL", "http://127.0.0.1:8080/v1"),
		transcriptionModel: envOr("TRANSCRIPTION_MODEL", "whisper-small"),
		analysisModel:      envOr("LLM_MODEL", "qwen3-4b"),
		maxAudioBytes:      maxAudioBytes, processingTimeout: timeout, timezone: location,
	}, nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
