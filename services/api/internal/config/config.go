package config

type Config struct {
	Database    Database    `mapstructure:"DB" json:"DB" yaml:"DB"`
	Redis       Redis       `mapstructure:"REDIS" json:"REDIS" yaml:"REDIS"`
	Application Application `mapstructure:"SERVER" json:"SERVER" yaml:"SERVER"`
	Github      Github      `mapstructure:"GITHUB" json:"GITHUB" yaml:"GITHUB"`
	Cron        Cron        `mapstructure:"CRON" json:"CRON" yaml:"CRON"`
	Log         Log         `mapstructure:"LOG" json:"LOG" yaml:"LOG"`
	Kafka       Kafka       `mapstructure:"KAFKA" json:"KAFKA" yaml:"KAFKA"`
	FrontendURL string      `mapstructure:"FRONTEND_URL" json:"FRONTEND_URL" yaml:"FRONTEND_URL"`
}

// Kafka configures the producer that publishes notification jobs. Brokers is a
// comma-separated list (a single address is the common case); Topic and
// DLQTopic default to the canonical names in ghnotify/contract.
type Kafka struct {
	Brokers  string `mapstructure:"BROKERS" json:"BROKERS" yaml:"BROKERS" default:"localhost:9092"`
	Topic    string `mapstructure:"TOPIC" json:"TOPIC" yaml:"TOPIC" default:"notifications.events"`
	DLQTopic string `mapstructure:"DLQ_TOPIC" json:"DLQ_TOPIC" yaml:"DLQ_TOPIC" default:"notifications.events.dlq"`
}

// Log configures structured application logging. The defaults emit JSON to
// stdout, which is what the Filebeat -> Elasticsearch -> Kibana pipeline
// consumes. Set ENCODING=console for human-readable local development.
type Log struct {
	Level       string `mapstructure:"LEVEL" json:"LEVEL" yaml:"LEVEL" default:"info"`
	Encoding    string `mapstructure:"ENCODING" json:"ENCODING" yaml:"ENCODING" default:"json"`
	Environment string `mapstructure:"ENVIRONMENT" json:"ENVIRONMENT" yaml:"ENVIRONMENT" default:"development"`
	ServiceName string `mapstructure:"SERVICE_NAME" json:"SERVICE_NAME" yaml:"SERVICE_NAME" default:"se-school"`
	Version     string `mapstructure:"VERSION" json:"VERSION" yaml:"VERSION" default:"dev"`
}

type Database struct {
	DNS string `mapstructure:"DSN" json:"DSN" yaml:"DSN"`
}

type Application struct {
	Port   string `mapstructure:"PORT" json:"PORT" yaml:"PORT"`
	APIKey string `mapstructure:"API_KEY" json:"API_KEY" yaml:"API_KEY"`
}

type Github struct {
	Token   string `mapstructure:"TOKEN" json:"TOKEN" yaml:"TOKEN"`
	BaseURL string `mapstructure:"BASE_URL" json:"BASE_URL" yaml:"BASE_URL"`
}

type Redis struct {
	Address  string `mapstructure:"ADDRESS" json:"ADDRESS" yaml:"ADDRESS" default:"localhost:6379"`
	Password string `mapstructure:"PASSWORD" json:"PASSWORD" yaml:"PASSWORD"`
	DB       int    `mapstructure:"DB" json:"DB" yaml:"DB" default:"0"`
}

type Cron struct {
	RepoCheckSchedule string `mapstructure:"REPO_CHECK_SCHEDULE" json:"REPO_CHECK_SCHEDULE" yaml:"REPO_CHECK_SCHEDULE" default:"0 * * * *"`
}
