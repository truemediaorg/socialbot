package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	Twitter   TwitterConfig
	Truemedia TruemediaConfig

	PostgresURL        string
	PostgresSecretPath string

	LogLevel        log.Level
	LogFormat       LogFormat
	TestModeEnabled bool
}

type TwitterConfig struct {
	BotUserName      string
	SecretPath       string
	TimelinePageSize int
}

type TruemediaConfig struct {
	ApiURL          url.URL
	ResolveInterval time.Duration
	ResultsInterval time.Duration
	SecretPath      string
}

type LogFormat string

const (
	LogFormatText = "text"
	LogFormatJSON = "json"
)

type EnvfileKey string

const (
	// Postgres connection string to use for database connections
	EnvfileKeyPostgresURL = "POSTGRES_URL"
	// AWS Secrets Manager path where Postgres connection string can be found
	EnvfileKeyPostgresSecretsPath = "POSTGRES_SECRETS_PATH"

	// Base URL to the Truemedia API, including "/api"
	EnvfileKeyTruemediaAPI = "TRUEMEDIA_API"
	// Interval to wait after calling the resolve media endpoint, in seconds
	EnvfileKeyTruemediaResolveInterval = "TRUEMEDIA_RESOLVE_INTERVAL"
	// Interval to wait after calling the get results endpoint, in seconds
	EnvfileKeyTruemediaResultsInterval = "TRUEMEDIA_RESULTS_INTERVAL"
	// AWS Secrets Manager path where Truemedia API secrets can be found
	EnvfileKeyTruemediaSecretPath = "TRUEMEDIA_SECRETS_PATH"

	// AWS Secrets Manager path where Twitter secrets can be found
	EnvfileKeyTwitterSecretPath = "TWITTER_SECRETS_PATH"
	// Twitter username of the bot, used for tracking mentions
	// NOTE: the bot posts under the account configured in twitter secrets
	EnvfileKeyTwitterUserName = "TWITTER_USERNAME"
	// Number of tweets to request per call to the timeline mentions endpoint
	EnvfileKeyTwitterTimelinePageSize = "TWITTER_TIMELINE_PAGE_SIZE"

	// Log level (e.g. "debug", "info", "warn", "error")
	EnvfileKeyLogLevel = "LOG_LEVEL"
	// Log output format (e.g. "text", "json")
	EnvfileKeyLogFormat = "LOG_FORMAT"
	// Enables "test mode" (server simulates posting, etc.)
	EnvfileKeyTestMode = "TEST_MODE"
)

func FromEnvfile() Config {
	viper.AddConfigPath(".")
	viper.SetConfigName(".env")
	viper.SetConfigType("dotenv")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	truemediaURL, err := url.Parse(getConfigString(EnvfileKeyTruemediaAPI))
	if err != nil {
		log.Fatalf("error parsing Truemedia URL: %v", err)
	}

	// TODO: Does this config make more sense as the actual user ID?
	twitterUsername := getConfigString(EnvfileKeyTwitterUserName)
	if twitterUsername == "" {
		log.Fatalf("must supply username for bot")
	}

	twitterTimelineSize := getConfigInt(EnvfileKeyTwitterTimelinePageSize)
	if twitterTimelineSize == 0 {
		// Default to 5 if not set
		twitterTimelineSize = 5
	}

	logLevel, err := log.ParseLevel(getConfigString(EnvfileKeyLogLevel))
	if err != nil {
		// Default to info level but log a warning
		log.Warnf("unable to parse log level: %v", err)
		logLevel = log.InfoLevel
	}

	logFormat, err := parseLogFormat(getConfigString(EnvfileKeyLogFormat))
	if err != nil {
		// Default to text formatter but log a warning
		log.Warnf("unable to parse log format: %v", err)
		logFormat = LogFormatText
	}

	postgresURL := getConfigString(EnvfileKeyPostgresURL)
	postgresSecretsPath := getConfigString(EnvfileKeyPostgresSecretsPath)
	if postgresURL == "" && postgresSecretsPath == "" {
		log.Fatal("postgres not configured")
	}

	isTestMode := viper.GetBool(EnvfileKeyTestMode)

	return Config{
		Truemedia: TruemediaConfig{
			ApiURL:          *truemediaURL,
			ResolveInterval: time.Duration(getConfigInt(EnvfileKeyTruemediaResolveInterval)) * time.Second,
			ResultsInterval: time.Duration(getConfigInt(EnvfileKeyTruemediaResultsInterval)) * time.Second,
			SecretPath:      getConfigString(EnvfileKeyTruemediaSecretPath),
		},
		Twitter: TwitterConfig{
			BotUserName:      twitterUsername,
			SecretPath:       getConfigString(EnvfileKeyTwitterSecretPath),
			TimelinePageSize: twitterTimelineSize,
		},
		PostgresURL:        postgresURL,
		PostgresSecretPath: postgresSecretsPath,
		LogLevel:           logLevel,
		LogFormat:          logFormat,
		TestModeEnabled:    isTestMode,
	}
}

func parseLogFormat(raw string) (LogFormat, error) {
	switch strings.ToLower(raw) {
	case LogFormatJSON:
		return LogFormatJSON, nil
	case LogFormatText:
		return LogFormatText, nil
	default:
		return "", fmt.Errorf("unidentified log format: %s", raw)
	}
}

// Gets a config value as a string from env vars or a .env file
func getConfigString(key string) string {
	value := os.Getenv(key)
	if value == "" {
		value = viper.GetString(key)
	}
	return value
}

// Gets a config value as an int from env vars or a .env file
func getConfigInt(key string) int {
	envVarValue := os.Getenv(key)
	if envVarValue == "" {
		return viper.GetInt(key)
	}
	value, err := strconv.Atoi(key)
	if err != nil {
		return 0
	}
	return value
}
