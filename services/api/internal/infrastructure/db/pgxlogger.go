package db

import (
	"context"
	"errors"

	"se-school/internal/infrastructure/logging"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
	"go.uber.org/zap"
)

// zapPgxLogger adapts pgx's tracelog.Logger onto our zap logger so that SQL
// queries and connection events are emitted as structured JSON instead of
// pgx's default plain text. This is the pgx replacement for the old GORM logger
// adapter; without it, query activity is invisible to the Filebeat ->
// Elasticsearch -> Kibana pipeline.
//
// It pulls the request-scoped logger from the context when present (set by the
// RequestID middleware), so SQL logs inherit the request_id of the HTTP request
// that triggered them. It is attached via pgxpool: poolCfg.ConnConfig.Tracer.
type zapPgxLogger struct{}

// Log implements tracelog.Logger. pgx emits successful queries/connects at
// LogLevelInfo and failures at LogLevelError; we map info/debug/trace down to
// zap debug so routine query chatter is hidden unless LOG_LEVEL=debug (matching
// the previous GORM logger), while warn/error always surface. Field names use
// ECS-style keys consistent with the rest of the logging package.
func (zapPgxLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	logger := logging.FromContext(ctx)

	fields := make([]zap.Field, 0, len(data))
	for k, v := range data {
		switch k {
		case "sql":
			fields = append(fields, zap.Any("db.statement", v))
		case "args":
			fields = append(fields, zap.Any("db.args", v))
		case "time":
			fields = append(fields, zap.Any("event.duration", v))
		case "rowCount":
			fields = append(fields, zap.Any("db.rows_affected", v))
		case "commandTag":
			fields = append(fields, zap.Any("db.command_tag", v))
		case "err":
			if err, ok := v.(error); ok && err != nil {
				fields = append(fields, zap.Error(err))
			}
		default:
			fields = append(fields, zap.Any(k, v))
		}
	}

	// pgx surfaces a no-rows result as an error; it is an expected, benign
	// outcome (e.g. confirm-code lookups), so don't log it at error level.
	if err, ok := data["err"].(error); ok && errors.Is(err, pgx.ErrNoRows) {
		level = tracelog.LogLevelDebug
	}

	switch level {
	case tracelog.LogLevelError:
		logger.Error(msg, fields...)
	case tracelog.LogLevelWarn:
		logger.Warn(msg, fields...)
	default:
		logger.Debug(msg, fields...)
	}
}
