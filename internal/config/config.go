package config

type Config struct {
	Database    Database    `mapstructure:"DB" json:"DB" yaml:"DB"`
	Application Application `mapstructure:"SERVER" json:"SERVER" yaml:"SERVER"`
	Github      Github      `mapstructure:"GITHUB" json:"GITHUB" yaml:"GITHUB"`
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
