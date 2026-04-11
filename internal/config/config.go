package config

type Config struct {
	Database    Database    `mapstructure:"DB" json:"DB" yaml:"DB"`
	Application Application `mapstructure:"SERVER" json:"SERVER" yaml:"SERVER"`
	Github      Github      `mapstructure:"GITHUB" json:"GITHUB" yaml:"GITHUB"`
	Mailer      Mailer      `mapstructure:"MAILER" json:"MAILER" yaml:"MAILER"`
}

type Database struct {
	DNS string `mapstructure:"DNS" json:"DNS" yaml:"DNS"`
}

type Application struct {
	Port string `mapstructure:"PORT" json:"PORT" yaml:"PORT"`
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
