package config

import (
	"github.com/phsym/console-slog"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"strings"
)

type AppConfig struct {
	App   App   `mapstructure:"app"`
	Kafka Kafka `mapstructure:"kafka"`
}

type Http struct {
	Auth Auth   `mapstructure:"auth"`
	Port string `mapstructure:"port"`
}

type App struct {
	Name     string `mapstructure:"name"`
	LogLevel string `mapstructure:"log-level"`
	Http     Http   `mapstructure:"http"`
}

type Kafka struct {
	BootstrapServers string `mapstructure:"bootstrap-servers"`
	OutputTopic      string `mapstructure:"output-topic"`
	SecurityProtocol string `mapstructure:"security-protocol"`
	Ssl              Ssl    `mapstructure:"ssl"`
}

type Ssl struct {
	CaLocation          string `mapstructure:"ca-location"`
	CertificateLocation string `mapstructure:"certificate-location"`
	KeyLocation         string `mapstructure:"key-location"`
	KeyPassword         string `mapstructure:"key-password"`
}

type Auth struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

func parseConfig(path string) (config *AppConfig, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("yml")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`, `-`, `_`))

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(&config)
	return config, err
}

func LoadConfig(path string) *AppConfig {
	c, err := parseConfig(path)
	if err != nil {
		slog.Error("Unable to load config file", "error", err)
		return nil
	}
	return c
}

func ConfigureLogger(c App) {
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelInfo)
	logger := slog.New(console.NewHandler(os.Stderr, &console.HandlerOptions{Level: lvl}))
	slog.SetDefault(logger)

	// set configured log level
	err := lvl.UnmarshalText([]byte(c.LogLevel))
	if err != nil {
		slog.Error("Unable to set Log level from application properties", "level", c.LogLevel, "error", err)
	}
}
