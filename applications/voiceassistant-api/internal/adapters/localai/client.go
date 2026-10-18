package localai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"voiceassistant-api/internal/domain"
)

type Config struct {
	BaseURL            string
	TranscriptionModel string
	AnalysisModel      string
	Timezone           *time.Location
	HTTPClient         *http.Client
}

type Client struct{ config Config }

func New(config Config) (*Client, error) {
	address, err := url.Parse(config.BaseURL)
	if err != nil || address.Host == "" || (address.Scheme != "http" && address.Scheme != "https") || address.User != nil || address.RawQuery != "" || address.Fragment != "" {
		return nil, errors.New("LocalAI base URL must be an HTTP or HTTPS URL without credentials, query, or fragment")
	}
	if config.TranscriptionModel == "" || config.AnalysisModel == "" || config.Timezone == nil || config.HTTPClient == nil {
		return nil, errors.New("LocalAI models, timezone, and HTTP client are required")
	}
	config.BaseURL = strings.TrimRight(config.BaseURL, "/")
	return &Client{config: config}, nil
}

func (c *Client) request(ctx context.Context, path, contentType string, body io.Reader, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+path, body)
	if err != nil {
		return fmt.Errorf("create LocalAI request: %w", err)
	}
	request.Header.Set("Content-Type", contentType)
	response, err := c.config.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("call LocalAI: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("LocalAI returned HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(target); err != nil {
		return fmt.Errorf("decode LocalAI response: %w", err)
	}
	return nil
}

var (
	_ domain.Transcriber = (*Client)(nil)
	_ domain.Analyzer    = (*Client)(nil)
)
