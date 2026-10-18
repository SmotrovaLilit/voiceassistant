package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"voiceassistant-api/internal/domain"
)

type AudioMessageRepository struct{ db *sql.DB }

func NewAudioMessageRepository(db *sql.DB) *AudioMessageRepository {
	return &AudioMessageRepository{db: db}
}

func (r *AudioMessageRepository) Create(ctx context.Context, message domain.AudioMessage) error {
	result, err := r.db.ExecContext(ctx, `INSERT INTO audiomessages
		(id, file_id, transcription, analysis, created_at)
		VALUES (?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`,
		message.ID, message.FileID, message.Transcription, message.Analysis,
		message.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert audio message: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read inserted row count: %w", err)
	}
	if count == 0 {
		return domain.ErrAlreadyExists
	}
	return nil
}

func (r *AudioMessageRepository) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM audiomessages WHERE id = ?)`, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("check audio message existence: %w", err)
	}
	return exists, nil
}

var _ domain.AudioMessageRepository = (*AudioMessageRepository)(nil)
