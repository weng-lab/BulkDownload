package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type loggerContextKey struct{}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func newRequestLoggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			requestLogger := logger.With(
				"method", r.Method,
				"path", r.URL.Path,
			)

			recorder := &statusRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
			}
			request := r.WithContext(context.WithValue(r.Context(), loggerContextKey{}, requestLogger))

			next.ServeHTTP(recorder, request)

			requestLogger.Info(
				"request finished",
				"status", recorder.status,
				"duration_ms", time.Since(startedAt).Milliseconds(),
			)
		})
	}
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func loggerFromContext(ctx context.Context) (*slog.Logger, bool) {
	logger, ok := ctx.Value(loggerContextKey{}).(*slog.Logger)
	return logger, ok
}

func requestLogger(r *http.Request) *slog.Logger {
	logger, ok := loggerFromContext(r.Context())
	if ok {
		return logger
	}

	return slog.Default()
}
