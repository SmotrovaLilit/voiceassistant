package application

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"voiceassistant-api/internal/domain"
)

type Service struct {
	messages    domain.AudioMessageRepository
	files       domain.FileRepository
	transcriber domain.Transcriber
	analyzer    domain.Analyzer
}

func NewService(messages domain.AudioMessageRepository, files domain.FileRepository, transcriber domain.Transcriber, analyzer domain.Analyzer) *Service {
	return &Service{messages: messages, files: files, transcriber: transcriber, analyzer: analyzer}
}

func (s *Service) UploadAudio(ctx context.Context, upload domain.AudioUpload) (domain.AudioMessage, error) {
	exists, err := s.messages.Exists(ctx, upload.ID)
	if err != nil {
		return domain.AudioMessage{}, fmt.Errorf("check audio ID: %w", err)
	}
	if exists {
		return domain.AudioMessage{}, domain.ErrAlreadyExists
	}

	now := time.Now().UTC()
	message := domain.AudioMessage{
		ID: upload.ID, FileID: rand.Text(),
		CreatedAt: now,
	}
	if err := s.files.Save(ctx, message.FileID, upload.Data); err != nil {
		return domain.AudioMessage{}, fmt.Errorf("save audio file: %w", err)
	}

	transcription, err := s.transcribe(ctx, upload, message.FileID)
	if err != nil {
		return domain.AudioMessage{}, fmt.Errorf("transcribe audio: %w", err)
	}
	message.Transcription = transcription
	analysis, err := s.analyzer.Analyze(ctx, transcription, message.CreatedAt)
	if err != nil {
		return domain.AudioMessage{}, fmt.Errorf("analyze audio: %w", err)
	}
	message.Analysis = analysis
	if err := s.messages.Create(ctx, message); err != nil {
		return domain.AudioMessage{}, fmt.Errorf("save audio message: %w", err)
	}
	return message, nil
}

func (s *Service) transcribe(ctx context.Context, upload domain.AudioUpload, fileID string) (string, error) {
	file, err := s.files.Open(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("open stored audio: %w", err)
	}
	upload.Data = file
	text, transcriptionErr := s.transcriber.Transcribe(ctx, upload)
	return text, errors.Join(transcriptionErr, file.Close())
}
