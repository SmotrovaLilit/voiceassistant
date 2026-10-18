package localai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"

	"voiceassistant-api/internal/domain"
)

func (c *Client) Transcribe(ctx context.Context, audio domain.AudioUpload) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": audio.Filename}))
	header.Set("Content-Type", audio.ContentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", fmt.Errorf("create transcription file part: %w", err)
	}
	if _, err := io.Copy(part, audio.Data); err != nil {
		return "", fmt.Errorf("read transcription audio: %w", err)
	}
	if err := writer.WriteField("model", c.config.TranscriptionModel); err != nil {
		return "", fmt.Errorf("write transcription model: %w", err)
	}
	if err := writer.WriteField("response_format", "json"); err != nil {
		return "", fmt.Errorf("write transcription format: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close transcription form: %w", err)
	}
	var response struct {
		Text string `json:"text"`
	}
	if err := c.request(ctx, "/audio/transcriptions", writer.FormDataContentType(), &body, &response); err != nil {
		return "", err
	}
	if strings.TrimSpace(response.Text) == "" {
		return "", errors.New("LocalAI returned an empty transcription")
	}
	return strings.TrimSpace(response.Text), nil
}
