package domain

import (
	"context"
	"errors"
	"io"
	"time"
)

var ErrAlreadyExists = errors.New("audio message already exists")

type AudioMessage struct {
	ID            string
	FileID        string
	Transcription string
	Analysis      string
	CreatedAt     time.Time
}

type AudioUpload struct {
	ID          string
	Filename    string
	ContentType string
	Data        io.Reader
}

type AudioMessageRepository interface {
	Create(ctx context.Context, message AudioMessage) error
	Exists(ctx context.Context, id string) (bool, error)
}

type FileRepository interface {
	Save(ctx context.Context, key string, content io.Reader) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}

type Transcriber interface {
	Transcribe(ctx context.Context, audio AudioUpload) (string, error)
}

type Analyzer interface {
	Analyze(ctx context.Context, transcription string, recordedAt time.Time) (string, error)
}
