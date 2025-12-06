package config

import (
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/goccy/go-yaml"
)

// Config holds the application configuration
type Config struct {
	App      AppConfig      `yaml:"app"`
	Server   ServerConfig   `yaml:"server"`
	Auth     AuthConfig     `yaml:"auth"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// AppConfig holds app-specific configuration
type AppConfig struct {
	Name string `yaml:"name"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// AuthConfig holds auth-specific configuration
type AuthConfig struct {
	KeysPath  string `yaml:"keys_path"`
	ActiveKID string `yaml:"active_kid"`
}

// DatabaseConfig holds database-specific configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// LoggingConfig holds logging-specific configuration
type LoggingConfig struct {
	Level string `yaml:"level"` // debug, info, warn, error
}

// Load reads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Address returns the server address in the format "host:port"
func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// quoteDSNValue quotes a DSN value if it contains spaces or special characters.
// Single quotes inside the value are escaped by doubling them.
func quoteDSNValue(value string) string {
	// Check if value needs quoting (contains spaces or special characters)
	needsQuoting := false
	for _, r := range value {
		if r == ' ' || r == '\'' || r == '\\' || r == '=' {
			needsQuoting = true
			break
		}
		// Quote if contains any non-alphanumeric character except common safe ones
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '.' || r == '-' || r == '_' || r == '/' || r == '@' || r == ':') {
			needsQuoting = true
			break
		}
	}

	if !needsQuoting {
		return value
	}

	// Escape single quotes by doubling them
	escaped := ""
	for _, r := range value {
		if r == '\'' {
			escaped += "''"
		} else {
			escaped += string(r)
		}
	}

	return "'" + escaped + "'"
}

// DSN returns the database connection string
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		quoteDSNValue(d.Host),
		d.Port,
		quoteDSNValue(d.User),
		quoteDSNValue(d.Password),
		quoteDSNValue(d.DBName),
		quoteDSNValue(d.SSLMode),
	)
}

// URL returns the database connection URL in postgres:// format for golang-migrate
func (d *DatabaseConfig) URL() string {
	// Use url.UserPassword to properly percent-encode username and password
	userInfo := url.UserPassword(d.User, d.Password)

	// Use net.JoinHostPort to properly handle IPv6 addresses (wraps them in brackets)
	host := net.JoinHostPort(d.Host, fmt.Sprintf("%d", d.Port))

	// Build URL with proper encoding
	u := &url.URL{
		Scheme:   "postgres",
		User:     userInfo,
		Host:     host,
		Path:     "/" + d.DBName,
		RawQuery: fmt.Sprintf("sslmode=%s&search_path=public", url.QueryEscape(d.SSLMode)),
	}

	return u.String()
}
