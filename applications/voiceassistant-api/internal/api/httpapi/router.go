package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	server "contracts/voiceassistant-api/pkg/server"
	"voiceassistant-api/internal/application"
)

const multipartOverhead = 1 << 20

type Config struct {
	Service           *application.Service
	MaxAudioBytes     int64
	ProcessingTimeout time.Duration
	Logger            *slog.Logger
}

type handler struct{ config Config }

func NewRouter(config Config) *gin.Engine {
	router := gin.New()
	router.ContextWithFallback = true
	router.Use(func(c *gin.Context) {
		started := time.Now()
		c.Next()
		if len(c.Errors) > 0 {
			config.Logger.ErrorContext(c.Request.Context(), "HTTP request failed",
				"method", c.Request.Method, "route", c.FullPath(), "status", c.Writer.Status(), "error", c.Errors.Last().Err)
		} else {
			config.Logger.InfoContext(c.Request.Context(), "HTTP request completed",
				"method", c.Request.Method, "route", c.FullPath(), "status", c.Writer.Status(), "duration", time.Since(started))
		}
	})
	router.Use(recovery())
	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, config.MaxAudioBytes+multipartOverhead)
		if c.Request.Method == http.MethodPost && c.FullPath() == "/v1/audiomessage" {
			mediaType, parameters, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
			if err != nil || mediaType != "multipart/form-data" {
				c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, server.Error{Code: "unsupported_media_type", Message: "Use multipart/form-data with id and audio fields."})
				return
			}
			if parameters["boundary"] == "" {
				c.AbortWithStatusJSON(http.StatusBadRequest, server.Error{Code: "invalid_multipart", Message: "The multipart boundary is missing."})
				return
			}
		}
		c.Next()
		if len(c.Errors) > 0 && !c.Writer.Written() {
			c.JSON(http.StatusInternalServerError, server.Error{Code: "internal_error", Message: "The request could not be processed."})
		}
	})
	server.RegisterHandlers(router, server.NewStrictHandler(&handler{config: config}, nil))
	return router
}

func recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recover() != nil {
				_ = c.Error(errors.New("HTTP handler panicked"))
				c.AbortWithStatusJSON(http.StatusInternalServerError, server.Error{Code: "internal_error", Message: "The request could not be processed."})
			}
		}()
		c.Next()
	}
}

func recordError(ctx context.Context, err error) {
	if c, ok := ctx.(*gin.Context); ok {
		_ = c.Error(err)
	}
}

type problemResponse struct {
	status int
	body   server.Error
}

func problem(status int, code, message string) problemResponse {
	return problemResponse{status: status, body: server.Error{Code: code, Message: message}}
}

func (p problemResponse) write(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(p.status)
	return json.NewEncoder(w).Encode(p.body)
}

func (p problemResponse) VisitUploadAudioMessageResponse(w http.ResponseWriter) error {
	return p.write(w)
}

var _ server.StrictServerInterface = (*handler)(nil)
