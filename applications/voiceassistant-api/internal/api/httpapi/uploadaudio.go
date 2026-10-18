package httpapi

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	server "contracts/voiceassistant-api/pkg/server"
	"voiceassistant-api/internal/domain"
)

func (h *handler) UploadAudioMessage(ctx context.Context, request server.UploadAudioMessageRequestObject) (server.UploadAudioMessageResponseObject, error) {
	if request.Body == nil {
		return problem(http.StatusBadRequest, "invalid_multipart", "The audio upload is missing."), nil
	}
	form, err := request.Body.ReadForm(8 << 20)
	if err != nil {
		var limit *http.MaxBytesError
		if errors.As(err, &limit) || errors.Is(err, multipart.ErrMessageTooLarge) {
			return problem(http.StatusRequestEntityTooLarge, "audio_too_large", "The audio exceeds the upload size limit."), nil
		}
		return problem(http.StatusBadRequest, "invalid_multipart", "The multipart request could not be read."), nil
	}
	defer func() {
		if err := form.RemoveAll(); err != nil {
			recordError(ctx, err)
		}
	}()
	upload, issue := h.parseUpload(form)
	if issue != nil {
		return *issue, nil
	}
	file, err := form.File["audio"][0].Open()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			recordError(ctx, err)
		}
	}()
	upload.Data = file
	processingCtx, cancel := context.WithTimeout(ctx, h.config.ProcessingTimeout)
	defer cancel()
	message, err := h.config.Service.UploadAudio(processingCtx, upload)
	if errors.Is(err, domain.ErrAlreadyExists) {
		return problem(http.StatusConflict, "audiomessage_already_exists", "An audio message with this UUID already exists."), nil
	}
	if err != nil {
		recordError(ctx, err)
		return problem(http.StatusServiceUnavailable, "processing_unavailable", "The audio could not be processed."), nil
	}
	response, err := convertAudioMessageToResponse(message)
	if err != nil {
		return nil, err
	}
	return server.UploadAudioMessage201JSONResponse(response), nil
}

func (h *handler) parseUpload(form *multipart.Form) (domain.AudioUpload, *problemResponse) {
	if len(form.Value["id"]) != 1 || len(form.File["audio"]) != 1 {
		issue := problem(http.StatusBadRequest, "invalid_upload", "Provide exactly one id field and one audio file.")
		return domain.AudioUpload{}, &issue
	}
	id, err := uuid.Parse(strings.TrimSpace(form.Value["id"][0]))
	if err != nil || id == uuid.Nil {
		issue := problem(http.StatusBadRequest, "invalid_id", "A non-zero audio message UUID is required.")
		return domain.AudioUpload{}, &issue
	}
	file := form.File["audio"][0]
	if file.Size <= 0 {
		issue := problem(http.StatusBadRequest, "audio_empty", "The audio file must not be empty.")
		return domain.AudioUpload{}, &issue
	}
	if file.Size > h.config.MaxAudioBytes {
		issue := problem(http.StatusRequestEntityTooLarge, "audio_too_large", "The audio exceeds the upload size limit.")
		return domain.AudioUpload{}, &issue
	}
	contentType := audioContentType(file.Filename)
	if contentType == "" {
		issue := problem(http.StatusUnsupportedMediaType, "unsupported_audio_format", "Use FLAC, MP3, MP4, MPEG, MPGA, M4A, OGG, WAV, or WebM.")
		return domain.AudioUpload{}, &issue
	}
	if declared, _, err := mime.ParseMediaType(file.Header.Get("Content-Type")); err == nil &&
		(strings.HasPrefix(declared, "audio/") || declared == "video/mp4" || declared == "video/webm") {
		contentType = declared
	}
	return domain.AudioUpload{
		ID: id.String(), Filename: filepath.Base(file.Filename), ContentType: contentType,
	}, nil
}

func audioContentType(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".flac":
		return "audio/flac"
	case ".mp3", ".mpeg", ".mpga":
		return "audio/mpeg"
	case ".m4a", ".mp4":
		return "audio/mp4"
	case ".ogg":
		return "audio/ogg"
	case ".wav":
		return "audio/wav"
	case ".webm":
		return "audio/webm"
	default:
		return ""
	}
}

func convertAudioMessageToResponse(message domain.AudioMessage) (server.AudioMessage, error) {
	id, err := uuid.Parse(message.ID)
	if err != nil {
		return server.AudioMessage{}, fmt.Errorf("decode audio message UUID: %w", err)
	}
	return server.AudioMessage{
		Id: id, Transcription: message.Transcription, Analysis: message.Analysis,
		CreatedAt: message.CreatedAt,
	}, nil
}
