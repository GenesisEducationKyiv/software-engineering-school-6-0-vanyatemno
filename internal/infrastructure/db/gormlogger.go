package db

import (
	"context"
	"errors"
	"time"

	"se-school/internal/infrastructure/logging"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// zapGormLogger adapts GORM's logger.Interface onto our zap logger so that SQL
// queries and ORM errors are emitted as structured JSON instead of GORM's
// default colorized, multi-line plain text. Without this, Filebeat fails to
// decode GORM's output (ANSI escape codes, line breaks) and the records show up
// in Elasticsearch as "Error decoding JSON" rather than queryable documents.
//
// It pulls the request-scoped logger from the context when present, so SQL logs
// inherit the request_id of the HTTP request that triggered them.
type zapGormLogger struct {
	level gormlogger.LogLevel
}

// newGormLogger returns a GORM logger backed by zap. Normal queries are logged
// at debug (set LOG_LEVEL=debug to see them); slow/erroring queries surface at
// warn/error regardless.
func newGormLogger() gormlogger.Interface {
	return &zapGormLogger{level: gormlogger.Info}
}

func (l *zapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	clone := *l
	clone.level = level
	return &clone
}

func (l *zapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= gormlogger.Info {
		logging.FromContext(ctx).Sugar().Infof(msg, data...)
	}
}

func (l *zapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= gormlogger.Warn {
		logging.FromContext(ctx).Sugar().Warnf(msg, data...)
	}
}

func (l *zapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= gormlogger.Error {
		logging.FromContext(ctx).Sugar().Errorf(msg, data...)
	}
}

// Trace logs one record per executed SQL statement using ECS-style field names.
func (l *zapGormLogger) Trace(
	ctx context.Context,
	begin time.Time,
	fc func() (sql string, rowsAffected int64),
	err error,
) {
	if l.level <= gormlogger.Silent {
		return
	}

	sql, rows := fc()
	fields := []zap.Field{
		zap.String("db.statement", sql),
		zap.Int64("db.rows_affected", rows),
		zap.Duration("event.duration", time.Since(begin)),
	}

	logger := logging.FromContext(ctx)

	switch {
	// ErrRecordNotFound is an expected, benign outcome (e.g. confirm-code
	// lookups); don't log it as an error.
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && l.level >= gormlogger.Error:
		logger.Error("sql query failed", append(fields, zap.Error(err))...)
	case l.level >= gormlogger.Info:
		logger.Debug("sql query", fields...)
	}
}
