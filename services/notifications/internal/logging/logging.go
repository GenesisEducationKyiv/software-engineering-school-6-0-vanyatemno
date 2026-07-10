// Package logging builds the notifier's structured logger: JSON to stdout with
// Elastic Common Schema field names for the shared Filebeat/Elasticsearch/Kibana
// pipeline. Set LOG_ENCODING=console for human-readable local output.
package logging

import (
	"os"

	"ghnotify/notifier/internal/config"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const encodingConsole = "console"

// Init attaches service metadata to every record so logs are attributable to a
// service/version/environment in Kibana.
func Init(cfg *config.Log) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid log level %q", cfg.Level)
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "@timestamp",
		LevelKey:       "log.level",
		NameKey:        "log.logger",
		CallerKey:      "log.origin",
		MessageKey:     "message",
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
