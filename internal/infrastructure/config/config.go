package config

import "os"

type Config struct {
	AppPort   string
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
	DBSSLMode string

	KafkaBroker  string
	KafkaGroupID string

	// Security
	APIKeys        string // Comma-separated list of valid API keys. Empty = auth disabled.
	AllowedOrigins string // Comma-separated CORS origins. "*" = allow all.
	RateLimitRPS   int    // Requests per second per IP (0 = disabled).
}

func Load() Config {
	return Config{
		AppPort:   getEnv("APP_PORT", "8000"),
		DBHost:    getEnv("DB_HOST", "go_db"),
		DBPort:    getEnv("DB_PORT", "5432"),
		DBUser:    getEnv("DB_USER", "postgres"),
		DBPass:    getEnv("DB_PASSWORD", "1234"),
		DBName:    getEnv("DB_NAME", "postgres"),
		DBSSLMode: getEnv("DB_SSLMODE", "disable"),

		KafkaBroker:  getEnv("KAFKA_BROKER", "kafka:9092"),
		KafkaGroupID: getEnv("KAFKA_GROUP_ID", "pix-consumer-group"),

		APIKeys:        getEnv("API_KEYS", ""),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
		RateLimitRPS:   getEnvInt("RATE_LIMIT_RPS", 60),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		var n int
		if _, err := parseIntFromString(v, &n); err == nil {
			return n
		}
	}
	return fallback
}

func parseIntFromString(s string, out *int) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errBadInt
		}
		n = n*10 + int(c-'0')
	}
	*out = n
	return n, nil
}

type intParseError string

func (e intParseError) Error() string { return string(e) }

const errBadInt intParseError = "not an integer"
