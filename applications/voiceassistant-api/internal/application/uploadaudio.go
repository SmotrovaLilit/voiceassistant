package application

import (
	"context"
	"fmt"
	"time"

	"voiceassistant-api/internal/domain"
)

type Service struct {
	crm         domain.CRMRepository
	transcriber domain.Transcriber
	analyzer    domain.Analyzer
}

func NewService(crm domain.CRMRepository, transcriber domain.Transcriber, analyzer domain.Analyzer) *Service {
	return &Service{crm: crm, transcriber: transcriber, analyzer: analyzer}
}

func (s *Service) UploadAudio(ctx context.Context, upload domain.AudioUpload) (domain.AudioMessage, error) {
	referenceTime := time.Now().UTC()
	transcription, err := s.transcriber.Transcribe(ctx, upload)
	if err != nil {
		return domain.AudioMessage{}, fmt.Errorf("transcribe audio: %w", err)
	}
	workflows, err := s.analyzer.Analyze(ctx, transcription, referenceTime)
	if err != nil {
		return domain.AudioMessage{}, fmt.Errorf("analyze audio: %w", err)
	}
	if err := s.crm.SaveWorkflows(ctx, workflows); err != nil {
		return domain.AudioMessage{}, fmt.Errorf("save workflows in CRM: %w", err)
	}
	return domain.AudioMessage{Transcription: transcription, Workflows: workflows}, nil
}
