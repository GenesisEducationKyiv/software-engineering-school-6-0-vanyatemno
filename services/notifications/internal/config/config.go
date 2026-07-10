// Package config loads the notifications service configuration from the
// environment (and an optional .env file). It deliberately reads only the
// subset of settings the notifier needs — Redis (to subscribe), Mailer (to
// send) and Log — sharing the same env var names as the API service so a single
// .env file can drive the whole cluster.
package config

import (
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	viperDefaultDelimiter = "."
	defaultTagName        = "default"
	squashTagValue        = ",squash"
	mapStructureTagName   = "mapstructure"
	defaultEnvFileName    = ".env"
)

type Config struct {
	Kafka    Kafka    `mapstructure:"KAFKA"`
	Redis    Redis    `mapstructure:"REDIS"`
	Mailer   Mailer   `mapstructure:"MAILER"`
	Database Database `mapstructure:"DB"`
	GRPC     GRPC     `mapstructure:"GRPC"`
	Log      Log      `mapstructure:"LOG"`

	// DedupTTL bounds how long per-recipient idempotency markers live in Redis.
	// It must comfortably exceed any redelivery/republish window.
	DedupTTL time.Duration `mapstructure:"DEDUP_TTL" default:"168h"`
}

// GRPC configures the synchronous delivery server the API calls for
// repository-update notifications. Port is a net.Listen address, e.g. ":9090".
type GRPC struct {
	Port string `mapstructure:"PORT" default:":9090"`
}

// Kafka configures the consumer. Brokers is a comma-separated list (a single
// address is the common case); GroupID is the consumer group; MaxRetries bounds
// transient-failure retries before a message is dead-lettered. RepliesTopic is
// where the worker publishes saga replies back to the API orchestrator.
type Kafka struct {
	Brokers      string `mapstructure:"BROKERS" default:"localhost:9092"`
	Topic        string `mapstructure:"TOPIC" default:"notifications.events"`
	DLQTopic     string `mapstructure:"DLQ_TOPIC" default:"notifications.events.dlq"`
	RepliesTopic string `mapstructure:"REPLIES_TOPIC" default:"notifications.replies"`
	GroupID      string `mapstructure:"GROUP_ID" default:"notifications-service"`
	MaxRetries   int    `mapstructure:"MAX_RETRIES" default:"3"`
}

// Database configures the notifier's own Postgres, where it records the durable
// delivery state that makes it a genuine saga participant. DSN is empty by
// default so a misconfiguration fails fast at startup.
type Database struct {
	DSN string `mapstructure:"DSN" default:""`
}

type Redis struct {
	Address  string `mapstructure:"ADDRESS" default:"localhost:6379"`
	Password string `mapstructure:"PASSWORD"`
	DB       int    `mapstructure:"DB" default:"0"`
}

type Mailer struct {
	Host     string `mapstructure:"HOST"`
	Port     int    `mapstructure:"PORT"`
	Username string `mapstructure:"USERNAME"`
	From     string `mapstructure:"FROM"`
	SMTP     string `mapstructure:"SMTP"`
	Password string `mapstructure:"PASSWORD"`
}

type Log struct {
	Level       string `mapstructure:"LEVEL" default:"info"`
	Encoding    string `mapstructure:"ENCODING" default:"json"`
	Environment string `mapstructure:"ENVIRONMENT" default:"development"`
	ServiceName string `mapstructure:"SERVICE_NAME" default:"se-school-notifier"`
	Version     string `mapstructure:"VERSION" default:"dev"`
}

func Read() (*Config, error) {
	var cfg Config
	err := read(&cfg)

	return &cfg, err
}

func read(config any, opts ...viper.DecoderConfigOption) error {
	reader := viper.New()
	reader.SetEnvKeyReplacer(strings.NewReplacer(viperDefaultDelimiter, "_"))

	if _, err := os.Stat(defaultEnvFileName); !os.IsNotExist(err) {
		if errLoad := godotenv.Load(defaultEnvFileName); errLoad != nil {
			return errors.Wrap(errLoad, "read config")
		}
	}

	reader.AutomaticEnv()
	reader.SetTypeByDefaultValue(true)
	if err := setDefaults("", reader, reflect.StructField{}, reflect.ValueOf(config).Elem()); err != nil {
		return errors.WithMessage(err, "failed to apply defaults")
	}
	if err := reader.Unmarshal(config, opts...); err != nil {
		return errors.WithMessage(err, "failed to parse configuration")
	}

	return nil
}

// setDefaults sets default values for struct fields based on the `default` tag.
func setDefaults(parentName string, vip *viper.Viper, t reflect.StructField, v reflect.Value) error {
	if v.Kind() == reflect.Struct {
		value, ok := t.Tag.Lookup(mapStructureTagName)
		if ok && value != squashTagValue {
			if parentName != "" {
				parentName += viperDefaultDelimiter
			}
			parentName += strings.ToUpper(value)
		}
		for i := 0; i < v.NumField(); i++ {
			if err := setDefaults(parentName, vip, v.Type().Field(i), v.Field(i)); err != nil {
				return err
			}
		}

		return nil
	}
	value, _ := t.Tag.Lookup(defaultTagName)
	fieldName, ok := t.Tag.Lookup(mapStructureTagName)

	if ok && fieldName != squashTagValue {
		if parentName != "" {
			fieldName = parentName + viperDefaultDelimiter + strings.ToUpper(fieldName)
		}
		vip.SetDefault(strings.ToUpper(fieldName), value)
	}

	return nil
}
