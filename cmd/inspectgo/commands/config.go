package commands

import (
	"errors"

	"github.com/spf13/viper"
)

type Config struct {
	Dataset   string          `mapstructure:"dataset"`
	Scorer    string          `mapstructure:"scorer"`
	Workers   int             `mapstructure:"workers"`
	Output    string          `mapstructure:"output"`
	Format    string          `mapstructure:"format"`
	LogDir    string          `mapstructure:"log_dir"`
	LogFormat string          `mapstructure:"log_format"`
	Provider  string          `mapstructure:"provider"`
	Model     ModelConfig     `mapstructure:"model"`
	OpenAI    OpenAIConfig    `mapstructure:"openai"`
	Anthropic AnthropicConfig `mapstructure:"anthropic"`
}

type ModelConfig struct {
	Name         string `mapstructure:"name"`
	MockResponse string `mapstructure:"mock_response"`
}

type OpenAIConfig struct {
	Model          string `mapstructure:"model"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
	MaxRetries     int    `mapstructure:"max_retries"`
	BackoffMillis  int    `mapstructure:"backoff_millis"`
}

type AnthropicConfig struct {
	Model          string `mapstructure:"model"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
	MaxRetries     int    `mapstructure:"max_retries"`
	BackoffMillis  int    `mapstructure:"backoff_millis"`
	MaxTokens      int    `mapstructure:"max_tokens"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{}
	v := viper.New()
	v.SetConfigType("yaml")
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName(".inspectgo")
		v.AddConfigPath(".")
	}

	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
