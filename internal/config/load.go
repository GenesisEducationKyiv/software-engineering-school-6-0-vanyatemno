package config

import (
	"os"
	"reflect"
	"strings"

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

func Read() (*Config, error) {
	var cfg Config
	err := read(&cfg)

	return &cfg, err
}

func read(config any, opts ...viper.DecoderConfigOption) error {
	reader := viper.New()
	// replace default viper delimiter for env vars
	// does not affect .env files, for them default delimiter is expected
	reader.SetEnvKeyReplacer(strings.NewReplacer(viperDefaultDelimiter, "_"))

	if _, err := os.Stat(defaultEnvFileName); !os.IsNotExist(err) {
		// Loads env vars from file and sets them using syscall.Setenv
		errLoad := godotenv.Load(defaultEnvFileName)
		if errLoad != nil {
			return errors.Wrap(errLoad, "read config")
		}
	}

	reader.AutomaticEnv()
	reader.SetTypeByDefaultValue(true)
	err := setDefaults("", reader, reflect.StructField{}, reflect.ValueOf(config).Elem())
	if err != nil {
		return errors.WithMessage(err, "failed to apply defaults")
	}
	err = reader.Unmarshal(config, opts...)
	if err != nil {
		return errors.WithMessage(err, "failed to parse configuration")
	}

	return nil
}

// setDefaults - sets default values for struct fields based using value from default tag
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
