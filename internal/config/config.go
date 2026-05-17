package config

type Config struct {
	Database    Database    `mapstructure:"DB" json:"DB" yaml:"DB"`
	Redis       Redis       `mapstructure:"REDIS" json:"REDIS" yaml:"REDIS"`
	Application Application `mapstructure:"SERVER" json:"SERVER" yaml:"SERVER"`
	Github      Github      `mapstructure:"GITHUB" json:"GITHUB" yaml:"GITHUB"`
	Mailer      Mailer      `mapstructure:"MAILER" json:"MAILER" yaml:"MAILER"`
	Cron        Cron        `mapstructure:"CRON" json:"CRON" yaml:"CRON"`
	FrontendURL string      `mapstructure:"FRONTEND_URL" json:"FRONTEND_URL" yaml:"FRONTEND_URL"`
}

type Database struct {
	DNS string `mapstructure:"DSN" json:"DSN" yaml:"DSN"`
}

type Application struct {
	Port   string `mapstructure:"PORT" json:"PORT" yaml:"PORT"`
	APIKey string `mapstructure:"API_KEY" json:"API_KEY" yaml:"API_KEY"`
}

type Github struct {
	Token string `mapstructure:"TOKEN" json:"TOKEN" yaml:"TOKEN"`
}

type Mailer struct {
	Host     string `mapstructure:"HOST" json:"HOST" yaml:"HOST"`
	Port     int    `mapstructure:"PORT" json:"PORT" yaml:"PORT"`
	Username string `mapstructure:"USERNAME" json:"USERNAME" yaml:"USERNAME"`
	From     string `mapstructure:"FROM" yaml:"FROM"`
	SMTP     string `mapstructure:"SMTP" yaml:"SMTP"`
	Password string `mapstructure:"PASSWORD" yaml:"PASSWORD"`
}

type Redis struct {
	Address  string `mapstructure:"ADDRESS" json:"ADDRESS" yaml:"ADDRESS" default:"localhost:6379"`
	Password string `mapstructure:"PASSWORD" json:"PASSWORD" yaml:"PASSWORD"`
	DB       int    `mapstructure:"DB" json:"DB" yaml:"DB" default:"0"`
}

type Cron struct {
	RepoCheckSchedule string `mapstructure:"REPO_CHECK_SCHEDULE" json:"REPO_CHECK_SCHEDULE" yaml:"REPO_CHECK_SCHEDULE" default:"0 * * * *"`
}
