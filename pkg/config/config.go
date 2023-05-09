package config

import (
	"github.com/spf13/viper"
	"strings"
)

type AppConfig struct {
	App   App   `mapstructure:"app"`
	Kafka Kafka `mapstructure:"kafka"`
	Gics  Gics  `mapstructure:"gics"`
}

type App struct {
	Name     string `mapstructure:"name"`
	LogLevel string `mapstructure:"log-level"`
	Auth     Auth   `mapstructure:"auth"`
}

type Kafka struct {
	BootstrapServers string `mapstructure:"bootstrap-servers"`
	OutputTopics     string `mapstructure:"output-topic"`
	SecurityProtocol string `mapstructure:"security-protocol"`
	Ssl              Ssl    `mapstructure:"ssl"`
}

type Ssl struct {
	CaLocation          string `mapstructure:"ca-location"`
	CertificateLocation string `mapstructure:"certificate-location"`
	KeyLocation         string `mapstructure:"key-location"`
	KeyPassword         string `mapstructure:"key-password"`
}

type Gics struct {
	Endpoint string `mapstructure:"endpoint"`
	Auth     Auth   `mapstructure:"auth"`
	SignerId string `mapstructure:"signer-id"`
}

type Auth struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

func LoadConfig(path string) (config AppConfig, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("yml")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`, `-`, `_`))

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
