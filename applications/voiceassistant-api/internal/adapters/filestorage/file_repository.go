package filestorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"voiceassistant-api/internal/domain"
)

type FileRepository struct{ directory string }

func NewFileRepository(directory string) (*FileRepository, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create file storage: %w", err)
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve file storage: %w", err)
	}
	return &FileRepository{directory: absolute}, nil
}

func validKey(key string) bool {
	return key != "" && key != "." && key != ".." && !strings.ContainsAny(key, "/\\\x00") && !filepath.IsAbs(key)
}

func (r *FileRepository) Save(ctx context.Context, key string, content io.Reader) error {
	if !validKey(key) {
		return errors.New("invalid file key")
	}
	temporary, err := os.CreateTemp(r.directory, ".upload-*")
	if err != nil {
		return fmt.Errorf("create temporary audio file: %w", err)
	}
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporary.Name())
	}()
	if _, err := io.Copy(temporary, contextReader{ctx: ctx, reader: content}); err != nil {
		return fmt.Errorf("write audio file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync audio file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close audio file: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(temporary.Name(), filepath.Join(r.directory, key)); err != nil {
		return fmt.Errorf("commit audio file: %w", err)
	}
	return nil
}

func (r *FileRepository) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if !validKey(key) {
		return nil, errors.New("invalid file key")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(r.directory)
	if err != nil {
		return nil, fmt.Errorf("open file storage: %w", err)
	}
	defer func() { _ = root.Close() }()
	file, err := root.Open(key)
	if err != nil {
		return nil, fmt.Errorf("open audio file: %w", err)
	}
	return file, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}

var _ domain.FileRepository = (*FileRepository)(nil)
