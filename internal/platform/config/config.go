// Package config carga la configuración del servicio desde variables de entorno.
// Ver .env.example para la lista completa.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

// Config es la configuración resuelta del servicio.
type Config struct {
	Env      string // development | staging | production
	HTTP     HTTPConfig
	Database DatabaseConfig
	Supabase SupabaseConfig
	Async    AsyncConfig
	Log      LogConfig
}

type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

// SupabaseConfig agrupa lo necesario para verificar el JWT de Supabase (D-04)
// y para las llamadas puntuales a su API.
type SupabaseConfig struct {
	URL              string
	AnonKey          string
	ServiceRoleKey   string // NO usar en operación normal (D-02).
	JWKSURL          string
	JWTSecret        string // fallback HS256 legacy.
	ExpectedAudience string
	ExpectedIssuer   string
}

type AsyncConfig struct {
	Workers   int
	QueueSize int
}

type LogConfig struct {
	Level  string
	Format string // json | text
}

// Load lee el entorno y devuelve la configuración, o un error si falta algo obligatorio.
func Load() (Config, error) {
	cfg := Config{
		Env: getenv("APP_ENV", "development"),
		HTTP: HTTPConfig{
			Addr:            getenv("HTTP_ADDR", ":8080"),
			ReadTimeout:     getdur("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getdur("HTTP_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: getdur("HTTP_SHUTDOWN_TIMEOUT", 20*time.Second),
		},
		Database: DatabaseConfig{
			URL:             os.Getenv("DATABASE_URL"),
			MaxConns:        getint32("DATABASE_MAX_CONNS", 10),
			MinConns:        getint32("DATABASE_MIN_CONNS", 2),
			MaxConnLifetime: getdur("DATABASE_MAX_CONN_LIFETIME", time.Hour),
		},
		Supabase: SupabaseConfig{
			URL:              os.Getenv("SUPABASE_URL"),
			AnonKey:          os.Getenv("SUPABASE_ANON_KEY"),
			ServiceRoleKey:   os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
			JWKSURL:          os.Getenv("SUPABASE_JWKS_URL"),
			JWTSecret:        os.Getenv("SUPABASE_JWT_SECRET"),
			ExpectedAudience: getenv("JWT_EXPECTED_AUDIENCE", "authenticated"),
			ExpectedIssuer:   os.Getenv("JWT_EXPECTED_ISSUER"),
		},
		Async: AsyncConfig{
			Workers:   getint("ASYNC_WORKERS", 4),
			QueueSize: getint("ASYNC_QUEUE_SIZE", 256),
		},
		Log: LogConfig{
			Level:  getenv("LOG_LEVEL", "info"),
			Format: getenv("LOG_FORMAT", "json"),
		},
	}

	if cfg.Env != "development" && cfg.Database.URL == "" {
		return Config{}, fmt.Errorf("config: DATABASE_URL es obligatoria fuera de development")
	}
	return cfg, nil
}

// NewLogger construye el slog.Logger según la config.
func NewLogger(cfg Config) *slog.Logger {
	var level slog.Level
	switch cfg.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}
	if cfg.Log.Format == "text" {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getint(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// getint32 parsea acotando a 32 bits (lo que espera pgxpool para el tamaño del pool).
func getint32(key string, def int32) int32 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(n)
		}
	}
	return def
}

func getdur(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
