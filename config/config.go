package config

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	OAuth    OAuthConfig
	TOTP     TOTPConfig
	CORS     CORSConfig
	RateLimit RateLimitConfig
	GCP      GCPConfig
}

type ServerConfig struct {
	Port int
	Env  string
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	Addr     string
	Password string
}

type JWTConfig struct {
	AccessSecret   string
	AccessExpiry   time.Duration
	RefreshSecret  string
	RefreshExpiry  time.Duration
}

type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

type TOTPConfig struct {
	EncryptionKey string
	Issuer        string
}

type CORSConfig struct {
	AllowedOrigins []string
}

type RateLimitConfig struct {
	LoginLimit  string
	GlobalLimit string
}

type GCPConfig struct {
	ProjectID string
	Enabled   bool
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("ENV", "development")
	viper.SetDefault("JWT_ACCESS_EXPIRY", "15m")
	viper.SetDefault("JWT_REFRESH_EXPIRY", "168h")
	viper.SetDefault("TOTP_ISSUER", "AuthService")
	viper.SetDefault("RATE_LIMIT_LOGIN", "5-M")
	viper.SetDefault("RATE_LIMIT_GLOBAL", "100-M")

	_ = viper.ReadInConfig()

	cfg := &Config{
		Server: ServerConfig{
			Port: viper.GetInt("SERVER_PORT"),
			Env:  viper.GetString("ENV"),
		},
		Database: DatabaseConfig{
			URL: viper.GetString("DATABASE_URL"),
		},
		Redis: RedisConfig{
			Addr:     viper.GetString("REDIS_ADDR"),
			Password: viper.GetString("REDIS_PASSWORD"),
		},
		JWT: JWTConfig{
			AccessSecret:  viper.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),
		},
		OAuth: OAuthConfig{
			GoogleClientID:     viper.GetString("GOOGLE_CLIENT_ID"),
			GoogleClientSecret: viper.GetString("GOOGLE_CLIENT_SECRET"),
			GoogleRedirectURL:  viper.GetString("GOOGLE_REDIRECT_URL"),
		},
		TOTP: TOTPConfig{
			EncryptionKey: viper.GetString("TOTP_ENCRYPTION_KEY"),
			Issuer:        viper.GetString("TOTP_ISSUER"),
		},
		CORS: CORSConfig{
			AllowedOrigins: parseCORSOrigins(viper.GetString("CORS_ALLOWED_ORIGINS")),
		},
		RateLimit: RateLimitConfig{
			LoginLimit:  viper.GetString("RATE_LIMIT_LOGIN"),
			GlobalLimit: viper.GetString("RATE_LIMIT_GLOBAL"),
		},
	}

	if projectID := viper.GetString("GCP_PROJECT_ID"); projectID != "" {
		cfg.GCP.ProjectID = projectID
		cfg.GCP.Enabled = true

		if err := loadSecretsFromGCP(cfg); err != nil {
			return nil, fmt.Errorf("failed to load secrets from GCP: %w", err)
		}
	}

	var accessExpiry, refreshExpiry time.Duration
	var err error

	if accessExpiry, err = time.ParseDuration(viper.GetString("JWT_ACCESS_EXPIRY")); err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_EXPIRY: %w", err)
	}
	cfg.JWT.AccessExpiry = accessExpiry

	if refreshExpiry, err = time.ParseDuration(viper.GetString("JWT_REFRESH_EXPIRY")); err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRY: %w", err)
	}
	cfg.JWT.RefreshExpiry = refreshExpiry

	return cfg, nil
}

func loadSecretsFromGCP(cfg *Config) error {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create secret manager client: %w", err)
	}
	defer client.Close()

	secrets := map[string]*string{
		"db-url":                 &cfg.Database.URL,
		"redis-addr":             &cfg.Redis.Addr,
		"jwt-access-secret":      &cfg.JWT.AccessSecret,
		"jwt-refresh-secret":     &cfg.JWT.RefreshSecret,
		"google-client-id":       &cfg.OAuth.GoogleClientID,
		"google-client-secret":   &cfg.OAuth.GoogleClientSecret,
	}

	for secretName, target := range secrets {
		req := &secretmanagerpb.AccessSecretVersionRequest{
			Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", cfg.GCP.ProjectID, secretName),
		}

		result, err := client.AccessSecretVersion(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to access secret %s: %w", secretName, err)
		}

		*target = string(result.Payload.Data)
	}

	return nil
}

func parseCORSOrigins(originsStr string) []string {
	if originsStr == "" {
		return []string{}
	}
	var origins []string
	for _, origin := range strings.Split(originsStr, ",") {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
