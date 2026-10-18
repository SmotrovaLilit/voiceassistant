package domain

import (
	"context"
	"io"
	"time"
)

type AudioMessage struct {
	Transcription string
	Workflows     []Workflow
}

type AudioUpload struct {
	Filename    string
	ContentType string
	Data        io.Reader
}

type CRMRepository interface {
	SaveWorkflows(ctx context.Context, workflows []Workflow) error
}

type Transcriber interface {
	Transcribe(ctx context.Context, audio AudioUpload) (string, error)
}

type Analyzer interface {
	Analyze(ctx context.Context, transcription string, recordedAt time.Time) ([]Workflow, error)
}
