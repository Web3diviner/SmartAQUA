package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestParseDatabaseURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected DatabaseConfig
	}{
		{
			name: "standard postgresql URL",
			url:  "postgresql://user:password@host.railway.app:5432/dbname",
			expected: DatabaseConfig{
				Host:     "host.railway.app",
				Port:     5432,
				User:     "user",
				Password: "password",
				DBName:   "dbname",
				SSLMode:  "require",
			},
		},
		{
			name: "postgres URL with sslmode",
			url:  "postgres://myuser:mypass@localhost:5432/mydb?sslmode=disable",
			expected: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "myuser",
				Password: "mypass",
				DBName:   "mydb",
				SSLMode:  "disable",
			},
		},
		{
			name: "Railway format URL",
			url:  "postgresql://postgres:abc123@containers-us-west-123.railway.app:6543/railway",
			expected: DatabaseConfig{
				Host:     "containers-us-west-123.railway.app",
				Port:     6543,
				User:     "postgres",
				Password: "abc123",
				DBName:   "railway",
				SSLMode:  "require",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg DatabaseConfig
			err := parseDatabaseURL(&cfg, tt.url)
			if err != nil {
				t.Fatalf("parseDatabaseURL() error = %v", err)
			}

			if cfg.Host != tt.expected.Host {
				t.Errorf("Host = %v, want %v", cfg.Host, tt.expected.Host)
			}
			if cfg.Port != tt.expected.Port {
				t.Errorf("Port = %v, want %v", cfg.Port, tt.expected.Port)
			}
			if cfg.User != tt.expected.User {
				t.Errorf("User = %v, want %v", cfg.User, tt.expected.User)
			}
			if cfg.Password != tt.expected.Password {
				t.Errorf("Password = %v, want %v", cfg.Password, tt.expected.Password)
			}
			if cfg.DBName != tt.expected.DBName {
				t.Errorf("DBName = %v, want %v", cfg.DBName, tt.expected.DBName)
			}
		})
	}
}

func TestParseRedisURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected RedisConfig
	}{
		{
			name: "simple redis URL",
			url:  "redis://localhost:6379",
			expected: RedisConfig{
				Host: "localhost",
				Port: 6379,
			},
		},
		{
			name: "redis URL with password",
			url:  "redis://:mypassword@redis.railway.app:6379",
			expected: RedisConfig{
				Host:     "redis.railway.app",
				Port:     6379,
				Password: "mypassword",
			},
		},
		{
			name: "redis URL with db",
			url:  "redis://localhost:6379/2",
			expected: RedisConfig{
				Host: "localhost",
				Port: 6379,
				DB:   2,
			},
		},
		{
			name: "Railway redis URL",
			url:  "redis://default:abc123@containers-us-west-456.railway.app:6379",
			expected: RedisConfig{
				Host:     "containers-us-west-456.railway.app",
				Port:     6379,
				Password: "abc123",
			},
		},
		{
			name: "redis URL with db query",
			url:  "redis://localhost:6379?db=5",
			expected: RedisConfig{
				Host: "localhost",
				Port: 6379,
				DB:   5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg RedisConfig
			err := parseRedisURL(&cfg, tt.url)
			if err != nil {
				t.Fatalf("parseRedisURL() error = %v", err)
			}

			if cfg.Host != tt.expected.Host {
				t.Errorf("Host = %v, want %v", cfg.Host, tt.expected.Host)
			}
			if cfg.Port != tt.expected.Port {
				t.Errorf("Port = %v, want %v", cfg.Port, tt.expected.Port)
			}
			if cfg.DB != tt.expected.DB {
				t.Errorf("DB = %v, want %v", cfg.DB, tt.expected.DB)
			}
			if cfg.Password != tt.expected.Password {
				t.Errorf("Password = %v, want %v", cfg.Password, tt.expected.Password)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	viper.Reset()

	// Test that Load() works with defaults
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port == 0 {
		t.Error("Server.Port should have a default value")
	}

	if cfg.Database.Host == "" {
		t.Error("Database.Host should have a default value")
	}

	if cfg.Redis.Host == "" {
		t.Error("Redis.Host should have a default value")
	}
}

func TestLoadWithConnectionURLs(t *testing.T) {
	viper.Reset()

	t.Setenv("SFF_DATABASE_URL", "postgresql://dbuser:dbpass@db.example.com:5433/sff_prod?sslmode=require")
	t.Setenv("SFF_REDIS_URL", "redis://:redispass@redis.example.com:6380/2")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %v, want %v", cfg.Database.Host, "db.example.com")
	}
	if cfg.Database.Port != 5433 {
		t.Errorf("Database.Port = %v, want %v", cfg.Database.Port, 5433)
	}
	if cfg.Database.User != "dbuser" {
		t.Errorf("Database.User = %v, want %v", cfg.Database.User, "dbuser")
	}
	if cfg.Database.Password != "dbpass" {
		t.Errorf("Database.Password = %v, want %v", cfg.Database.Password, "dbpass")
	}
	if cfg.Database.DBName != "sff_prod" {
		t.Errorf("Database.DBName = %v, want %v", cfg.Database.DBName, "sff_prod")
	}
	if cfg.Database.SSLMode != "require" {
		t.Errorf("Database.SSLMode = %v, want %v", cfg.Database.SSLMode, "require")
	}

	if cfg.Redis.Host != "redis.example.com" {
		t.Errorf("Redis.Host = %v, want %v", cfg.Redis.Host, "redis.example.com")
	}
	if cfg.Redis.Port != 6380 {
		t.Errorf("Redis.Port = %v, want %v", cfg.Redis.Port, 6380)
	}
	if cfg.Redis.Password != "redispass" {
		t.Errorf("Redis.Password = %v, want %v", cfg.Redis.Password, "redispass")
	}
	if cfg.Redis.DB != 2 {
		t.Errorf("Redis.DB = %v, want %v", cfg.Redis.DB, 2)
	}
}

func TestLoadWithCloudinaryEnvVars(t *testing.T) {
	viper.Reset()

	t.Setenv("SFF_CLOUDINARY_CLOUD_NAME", "demo-cloud")
	t.Setenv("SFF_CLOUDINARY_API_KEY", "demo-key")
	t.Setenv("SFF_CLOUDINARY_API_SECRET", "demo-secret")
	t.Setenv("SFF_CLOUDINARY_FOLDER", "demo-folder")
	t.Setenv("SFF_CLOUDINARY_ENABLED", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Cloudinary.CloudName != "demo-cloud" {
		t.Errorf("Cloudinary.CloudName = %v, want %v", cfg.Cloudinary.CloudName, "demo-cloud")
	}
	if cfg.Cloudinary.APIKey != "demo-key" {
		t.Errorf("Cloudinary.APIKey = %v, want %v", cfg.Cloudinary.APIKey, "demo-key")
	}
	if cfg.Cloudinary.APISecret != "demo-secret" {
		t.Errorf("Cloudinary.APISecret = %v, want %v", cfg.Cloudinary.APISecret, "demo-secret")
	}
	if cfg.Cloudinary.Folder != "demo-folder" {
		t.Errorf("Cloudinary.Folder = %v, want %v", cfg.Cloudinary.Folder, "demo-folder")
	}
	if !cfg.Cloudinary.Enabled {
		t.Errorf("Cloudinary.Enabled = %v, want %v", cfg.Cloudinary.Enabled, true)
	}
}
