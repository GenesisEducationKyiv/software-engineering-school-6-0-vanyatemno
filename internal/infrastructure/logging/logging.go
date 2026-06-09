// Package logging builds the application's structured logger and provides
// helpers for propagating a request-scoped logger through context.Context.
//
// Logs are emitted as JSON to stdout using Elastic Common Schema (ECS)-friendly
// field names (@timestamp, log.level, message, service.*). This lets Filebeat
// collect the container's stdout, ship it to Elasticsearch without extra
// parsing, and makes the documents searchable/aggregatable in Kibana.
package logging

import (
	"context"
	"os"

	"se-school/internal/config"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// encodingConsole selects the human-readable encoder used for local development.
const encodingConsole = "console"

type ctxKey struct{}

// loggerCtxKey is the context key under which the request-scoped logger is stored.
var loggerCtxKey = ctxKey{}

// Init builds a *zap.Logger from configuration. The returned logger writes
// structured JSON (or console output when cfg.Encoding == "console") to stdout
// and is decorated with service metadata so every record is attributable to a
// service/version/environment in Kibana.
func Init(cfg *config.Log) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid log level %q", cfg.Level)
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:    "@timestamp",
		LevelKey:   "log.level",
		NameKey:    "log.logger",
		CallerKey:  "log.origin",
		MessageKey: "message",
		// Stacktraces live under a top-level "stacktrace" key, NOT under the
		// "error." namespace. zap.Error(err) emits a *scalar* "error" string, so
		// putting the stacktrace at "error.stack_trace" would make "error" both a
		// string and an object — which breaks Filebeat's json.expand_keys and
		// Elasticsearch's dotted-field mapping ("cannot expand error.stack_trace:
		// expected map but type is string"). Keeping it flat avoids that clash and
		// matches the Recovery middleware, which already logs zap.Stack("stacktrace").
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.Encoding == encodingConsole {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)

	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Fields(
			zap.String("service.name", cfg.ServiceName),
			zap.String("service.version", cfg.Version),
			zap.String("service.environment", cfg.Environment),
		),
	)

	return logger, nil
}

// ContextWithLogger returns a copy of ctx carrying the provided logger so that
// downstream code can retrieve a request-scoped logger via FromContext.
func ContextWithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey, logger)
}

// FromContext returns the logger stored in ctx by ContextWithLogger, falling
// back to the global logger (zap.L()) when none is present. It never returns nil.
func FromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerCtxKey).(*zap.Logger); ok && logger != nil {
		return logger
	}

	return zap.L()
}
