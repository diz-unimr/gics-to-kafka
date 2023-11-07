package config

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"runtime"
	"testing"
)

func TestLoadConfigWithEnv(t *testing.T) {
	setProjectDir()

	expected := "test"
	t.Setenv("KAFKA_OUTPUT_TOPIC", expected)

	config, _ := LoadConfig(".")

	assert.Equal(t, expected, config.Kafka.OutputTopic)
}

func TestLoadConfigFileNotFound(t *testing.T) {
	setProjectDir()

	// config file not found
	_, err := LoadConfig("./bla")

	if assert.Error(t, err) {
		_, isConfigError := err.(viper.ConfigFileNotFoundError)

		assert.True(t, isConfigError)
	}
}

func setProjectDir() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../..")
	_ = os.Chdir(dir)

	viper.Reset()
}
