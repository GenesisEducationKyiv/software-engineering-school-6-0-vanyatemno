// Package logging builds the app's structured logger and propagates a
// request-scoped logger through context. Logs are JSON to stdout with Elastic
// Common Schema field names for the Filebeat → Elasticsearch → Kibana pipeline.
package logging

import (
	"context"
	"os"

	"se-school/internal/config"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const encodingConsole = "console"

type ctxKey struct{}

var loggerCtxKey = ctxKey{}

// Init attaches service metadata to every record so logs are attributable to a
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

func ContextWithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey, logger)
}

// FromContext falls back to the global logger (zap.L()) when none is present; it
// never returns nil.
func FromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerCtxKey).(*zap.Logger); ok && logger != nil {
		return logger
	}

	return zap.L()
}
