package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppHost  string
	HTTPPort string
	GRPCPort string
	LogLevel string

	DB struct {
		Host     string
		Port     string
		User     string
		Password string
		Database string
		SSLMode  string
	}

	OperatorPoolURL    string        // e.g. http://localhost:8094
	SearchServiceURL   string        // опционально: URL search-service для индексации операторов (e.g. http://localhost:8096)
	KafkaBrokers       []string      // опционально: брокеры Kafka для событий операторов
	KafkaTopicOperator string        // топик для событий операторов (по умолчанию psds.operator.assigned)
	PoolTimeout        time.Duration // таймаут запроса к operator-pool
	PoolMaxRetries     int           // число повторов
	PoolRetryBackoffMs int           // пауза между повторами (мс)
}

func Load() (*Config, error) {
	cfg := &Config{
		AppHost:            getEnv("APP_HOST", "0.0.0.0"),
		HTTPPort:           firstEnv("APP_PORT", "HTTP_PORT", "8095"),
		GRPCPort:           firstEnv("GRPC_PORT", "METRICS_PORT", "9095"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		OperatorPoolURL:    getEnv("OPERATOR_POOL_URL", "http://localhost:8094"),
		SearchServiceURL:   getEnv("SEARCH_SERVICE_URL", ""),
		KafkaTopicOperator: getEnv("KAFKA_TOPIC_OPERATOR", "psds.operator.assigned"),
		PoolTimeout:        time.Duration(getEnvInt("POOL_TIMEOUT_SEC", 10)) * time.Second,
		PoolMaxRetries:     getEnvInt("POOL_MAX_RETRIES", 3),
		PoolRetryBackoffMs: getEnvInt("POOL_RETRY_BACKOFF_MS", 500),
	}
	if brokers := getEnv("KAFKA_BROKERS", ""); brokers != "" {
		for _, s := range strings.Split(brokers, ",") {
			if t := strings.TrimSpace(s); t != "" {
				cfg.KafkaBrokers = append(cfg.KafkaBrokers, t)
			}
		}
	}
	cfg.DB.Host = getEnv("DB_HOST", "localhost")
	cfg.DB.Port = getEnv("DB_PORT", "5432")
	cfg.DB.User = getEnv("DB_USER", "postgres")
	cfg.DB.Password = getEnv("DB_PASSWORD", "postgres")
	cfg.DB.Database = getEnv("DB_DATABASE", "operator_directory_service")
	cfg.DB.SSLMode = getEnv("DB_SSLMODE", "disable")
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.DB.Host == "" {
		return errors.New("config: DB_HOST is required")
	}
	if c.DB.User == "" {
		return errors.New("config: DB_USER is required")
	}
	if c.DB.Database == "" {
		return errors.New("config: DB_DATABASE is required")
	}
	if c.AppEnv() == "production" && c.DB.Password == "" {
		return errors.New("config: in production DB_PASSWORD is required")
	}
	return nil
}

func (c *Config) AppEnv() string {
	return getEnv("APP_ENV", "development")
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password, c.DB.Database, c.DB.SSLMode)
}

func (c *Config) DatabaseURL() string {
	pass := url.QueryEscape(c.DB.Password)
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DB.User, pass, c.DB.Host, c.DB.Port, c.DB.Database, c.DB.SSLMode)
}

func (c *Config) Addr() string {
	return c.AppHost + ":" + c.HTTPPort
}

func firstEnv(keysAndDef ...string) string {
	if len(keysAndDef) == 0 {
		return ""
	}
	def := keysAndDef[len(keysAndDef)-1]
	keys := keysAndDef[:len(keysAndDef)-1]
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return def
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
