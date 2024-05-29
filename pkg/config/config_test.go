package config

import (
	"context"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
	"path"
	"runtime"
	"testing"
)

func TestConfigureLoggerSetsLogLevel(t *testing.T) {
	setProjectDir()

	expected := "debug"

	ConfigureLogger(App{LogLevel: expected})

	assert.True(t, slog.Default().Enabled(context.Background(), slog.LevelDebug))
}

func TestParseConfigWithEnv(t *testing.T) {
	setProjectDir()

	expected := "test"
	t.Setenv("KAFKA_OUTPUT_TOPIC", expected)

	config, _ := parseConfig(".")

	assert.Equal(t, expected, config.Kafka.OutputTopic)
}

func TestParseConfigFileNotFound(t *testing.T) {
	setProjectDir()

	// config file not found
	_, err := parseConfig("./bla")

	if assert.Error(t, err) {
		_, isConfigError := err.(viper.ConfigFileNotFoundError)

		assert.True(t, isConfigError)
	}
}

func TestLoadDefaultConfig(t *testing.T) {
	setProjectDir()
	defaultConfig := AppConfig{
		App: App{
			Name:     "gics-to-kafka",
			LogLevel: "info",
			Http: Http{Port: "8080", Auth: Auth{
				User:     "test",
				Password: "test",
			}},
		},
		Kafka: Kafka{
			BootstrapServers: "localhost:9092",
			OutputTopic:      "gics-notification",
			SecurityProtocol: "ssl",
			Ssl: Ssl{
				CaLocation:          "/app/cert/kafka-ca.pem",
				CertificateLocation: "/app/cert/app-cert.pem",
				KeyLocation:         "/app/cert/app-key.pem",
			},
		},
	}
	actual := *LoadConfig(".")

	assert.Equal(t, defaultConfig, actual)
}

func TestLoadConfig_Error(t *testing.T) {
	setProjectDir()

	actual := LoadConfig("/")

	assert.Nil(t, actual)
}

func TestConfigureLogger(t *testing.T) {
	ConfigureLogger(App{LogLevel: "warn"})

	assert.True(t, slog.Default().Enabled(context.Background(), slog.LevelWarn))
	assert.False(t, slog.Default().Enabled(context.Background(), slog.LevelInfo))

	// invalid value defaults to info
	ConfigureLogger(App{LogLevel: "test"})
	assert.True(t, slog.Default().Enabled(context.Background(), slog.LevelInfo))
}

func setProjectDir() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../..")
	_ = os.Chdir(dir)

	viper.Reset()
}
