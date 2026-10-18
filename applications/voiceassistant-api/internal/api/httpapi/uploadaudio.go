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

	server "contracts/voiceassistant-api/pkg/server"
	"voiceassistant-api/internal/domain"
)

func (h *handler) UploadAudioMessage(ctx context.Context, request server.UploadAudioMessageRequestObject) (server.UploadAudioMessageResponseObject, error) {
	if request.Body == nil {
		return problem(http.StatusBadRequest, "invalid_multipart", "The audio upload is missing."), nil
	}
	form, err := request.Body.ReadForm(h.config.MaxAudioBytes + multipartOverhead)
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
	if err != nil {
		recordError(ctx, err)
		return problem(http.StatusServiceUnavailable, "processing_unavailable", "The audio could not be processed and saved in the CRM."), nil
	}
	response, err := convertAudioMessageToResponse(message)
	if err != nil {
		return nil, err
	}
	return server.UploadAudioMessage201JSONResponse(response), nil
}

func (h *handler) parseUpload(form *multipart.Form) (domain.AudioUpload, *problemResponse) {
	if len(form.File["audio"]) != 1 {
		issue := problem(http.StatusBadRequest, "invalid_upload", "Provide exactly one audio file.")
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
		Filename: filepath.Base(file.Filename), ContentType: contentType,
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
	workflows := make([]server.Workflow, 0, len(message.Workflows))
	for _, workflow := range message.Workflows {
		converted, err := convertWorkflowToResponse(workflow)
		if err != nil {
			return server.AudioMessage{}, err
		}
		workflows = append(workflows, converted)
	}
	return server.AudioMessage{
		Transcription: message.Transcription, Workflows: workflows,
	}, nil
}

func convertWorkflowToResponse(workflow domain.Workflow) (server.Workflow, error) {
	var response server.Workflow
	var err error
	switch payload := workflow.Payload.(type) {
	case domain.TaskPayload:
		err = response.FromTaskWorkflow(server.TaskWorkflow{
			Payload: server.TaskPayload{
				Title: payload.Title, ClientName: optionalString(payload.ClientName),
				Description: optionalString(payload.Description), DueAt: optionalString(payload.DueAt),
				RemindAt: optionalString(payload.RemindAt),
			},
		})
	case domain.VisitReportPayload:
		err = response.FromVisitReportWorkflow(server.VisitReportWorkflow{
			Payload: server.VisitReportPayload{
				Summary: payload.Summary, ClientName: optionalString(payload.ClientName),
				ContactName: optionalString(payload.ContactName), VisitedAt: optionalString(payload.VisitedAt),
			},
		})
	default:
		return response, fmt.Errorf("unsupported workflow payload %T", workflow.Payload)
	}
	if err != nil {
		return response, fmt.Errorf("encode workflow response: %w", err)
	}
	return response, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
