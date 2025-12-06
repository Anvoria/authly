package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabaseConfig_URL tests the new URL() method
func TestDatabaseConfig_URL(t *testing.T) {
	tests := []struct {
		name     string
		config   DatabaseConfig
		expected string
	}{
		{
			name: "standard configuration",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "testuser",
				Password: "testpass",
				DBName:   "testdb",
				SSLMode:  "disable",
			},
			expected: "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable&search_path=public",
		},
		{
			name: "with special characters in password",
			config: DatabaseConfig{
				Host:     "db.example.com",
				Port:     5433,
				User:     "admin",
				Password: "p@ss:w0rd!",
				DBName:   "production",
				SSLMode:  "require",
			},
			expected: "postgres://admin:p%40ss%3Aw0rd%21@db.example.com:5433/production?sslmode=require&search_path=public",
		},
		{
			name: "with IPv6 host",
			config: DatabaseConfig{
				Host:     "::1",
				Port:     5432,
				User:     "postgres",
				Password: "postgres",
				DBName:   "testdb",
				SSLMode:  "prefer",
			},
			expected: "postgres://postgres:postgres@[::1]:5432/testdb?sslmode=prefer&search_path=public",
		},
		{
			name: "with empty password",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "nopassuser",
				Password: "",
				DBName:   "testdb",
				SSLMode:  "disable",
			},
			expected: "postgres://nopassuser:@localhost:5432/testdb?sslmode=disable&search_path=public",
		},
		{
			name: "with non-standard port",
			config: DatabaseConfig{
				Host:     "remote-db",
				Port:     15432,
				User:     "remoteuser",
				Password: "remotepass",
				DBName:   "remotedb",
				SSLMode:  "verify-full",
			},
			expected: "postgres://remoteuser:remotepass@remote-db:15432/remotedb?sslmode=verify-full&search_path=public",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.URL()
			assert.Equal(t, tt.expected, result, "URL should match expected format")
		})
	}
}

// TestDatabaseConfig_DSN tests the existing DSN() method for consistency
func TestDatabaseConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		config   DatabaseConfig
		expected string
	}{
		{
			name: "standard configuration",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "testuser",
				Password: "testpass",
				DBName:   "testdb",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable",
		},
		{
			name: "with special characters",
			config: DatabaseConfig{
				Host:     "db.example.com",
				Port:     5433,
				User:     "admin",
				Password: "p@ss w0rd!",
				DBName:   "production",
				SSLMode:  "require",
			},
			expected: "host=db.example.com port=5433 user=admin password='p@ss w0rd!' dbname=production sslmode=require",
		},
		{
			name: "with single quotes in password",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "testuser",
				Password: "pass'word",
				DBName:   "testdb",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=testuser password='pass''word' dbname=testdb sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.DSN()
			assert.Equal(t, tt.expected, result, "DSN should match expected format")
		})
	}
}

// TestDatabaseConfig_URLAndDSN_Consistency tests that both methods produce valid outputs
func TestDatabaseConfig_URLAndDSN_Consistency(t *testing.T) {
	config := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		DBName:   "db",
		SSLMode:  "disable",
	}

	dsn := config.DSN()
	url := config.URL()

	assert.NotEmpty(t, dsn, "DSN should not be empty")
	assert.NotEmpty(t, url, "URL should not be empty")
	assert.Contains(t, url, "postgres://", "URL should start with postgres://")
	assert.Contains(t, url, "search_path=public", "URL should contain search_path parameter")
	assert.Contains(t, dsn, "host=", "DSN should contain host parameter")
	assert.Contains(t, dsn, "port=", "DSN should contain port parameter")
}

// TestServerConfig_Address tests the existing Address() method
func TestServerConfig_Address(t *testing.T) {
	tests := []struct {
		name     string
		config   ServerConfig
		expected string
	}{
		{
			name: "standard localhost",
			config: ServerConfig{
				Host: "0.0.0.0",
				Port: 8000,
			},
			expected: "0.0.0.0:8000",
		},
		{
			name: "custom host and port",
			config: ServerConfig{
				Host: "example.com",
				Port: 3000,
			},
			expected: "example.com:3000",
		},
		{
			name: "IPv6 address",
			config: ServerConfig{
				Host: "::1",
				Port: 8080,
			},
			expected: "::1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Address()
			assert.Equal(t, tt.expected, result, "Address should match expected format")
		})
	}
}

// TestLoad tests the Load function with valid and invalid YAML files
func TestLoad(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		// Create a temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `
app:
  name: "test-app"

server:
  host: "localhost"
  port: 8000

auth:
  keys_path: "keys"
  active_kid: "main"

database:
  host: "localhost"
  port: 5432
  user: "testuser"
  password: "testpass"
  dbname: "testdb"
  sslmode: "disable"

logging:
  level: "info"
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Load the config
		cfg, err := Load(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Verify all fields
		assert.Equal(t, "test-app", cfg.App.Name)
		assert.Equal(t, "localhost", cfg.Server.Host)
		assert.Equal(t, 8000, cfg.Server.Port)
		assert.Equal(t, "keys", cfg.Auth.KeysPath)
		assert.Equal(t, "main", cfg.Auth.ActiveKID)
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
		assert.Equal(t, "testuser", cfg.Database.User)
		assert.Equal(t, "testpass", cfg.Database.Password)
		assert.Equal(t, "testdb", cfg.Database.DBName)
		assert.Equal(t, "disable", cfg.Database.SSLMode)
		assert.Equal(t, "info", cfg.Logging.Level)
	})

	t.Run("non-existent file", func(t *testing.T) {
		cfg, err := Load("/nonexistent/path/config.yaml")
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("invalid yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid.yaml")

		invalidContent := `
app:
  name: "test-app"
  invalid: [unclosed array
`
		err := os.WriteFile(configPath, []byte(invalidContent), 0644)
		require.NoError(t, err)

		cfg, err := Load(configPath)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "empty.yaml")

		err := os.WriteFile(configPath, []byte(""), 0644)
		require.NoError(t, err)

		cfg, err := Load(configPath)
		// Empty file should parse successfully but have zero values
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("partial config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "partial.yaml")

		partialContent := `
app:
  name: "partial-app"
server:
  host: "localhost"
`
		err := os.WriteFile(configPath, []byte(partialContent), 0644)
		require.NoError(t, err)

		cfg, err := Load(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "partial-app", cfg.App.Name)
		assert.Equal(t, "localhost", cfg.Server.Host)
		assert.Equal(t, 0, cfg.Server.Port) // Default zero value
	})
}

// TestDatabaseConfig_URL_EdgeCases tests edge cases for URL generation
func TestDatabaseConfig_URL_EdgeCases(t *testing.T) {
	t.Run("empty fields", func(t *testing.T) {
		config := DatabaseConfig{}
		url := config.URL()
		assert.Contains(t, url, "postgres://")
		assert.Contains(t, url, "search_path=public")
	})

	t.Run("zero port", func(t *testing.T) {
		config := DatabaseConfig{
			Host:     "localhost",
			Port:     0,
			User:     "user",
			Password: "pass",
			DBName:   "db",
			SSLMode:  "disable",
		}
		url := config.URL()
		assert.Contains(t, url, ":0/")
	})

	t.Run("url special characters in database name", func(t *testing.T) {
		config := DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "pass",
			DBName:   "test-db_123",
			SSLMode:  "disable",
		}
		url := config.URL()
		assert.Contains(t, url, "test-db_123")
	})
}

// TestDatabaseConfig_URL_MigrationCompatibility tests URL format for golang-migrate
func TestDatabaseConfig_URL_MigrationCompatibility(t *testing.T) {
	t.Run("URL should be compatible with golang-migrate", func(t *testing.T) {
		config := DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			DBName:   "testdb",
			SSLMode:  "disable",
		}

		url := config.URL()

		// golang-migrate expects this format
		assert.Contains(t, url, "postgres://", "Should use postgres:// scheme")
		assert.Contains(t, url, "search_path=public", "Should specify search_path")
		assert.Contains(t, url, "sslmode=", "Should include sslmode")

		// Verify structure
		assert.Regexp(t, `^postgres://[^:]+:[^@]*@[^/]+/[^?]+\?.*$`, url, "Should match URL pattern")
	})

	t.Run("URL should preserve all connection parameters", func(t *testing.T) {
		config := DatabaseConfig{
			Host:     "prod-db.example.com",
			Port:     5433,
			User:     "produser",
			Password: "prodpass",
			DBName:   "production",
			SSLMode:  "require",
		}

		url := config.URL()

		assert.Contains(t, url, "produser", "Should contain username")
		assert.Contains(t, url, "prodpass", "Should contain password")
		assert.Contains(t, url, "prod-db.example.com", "Should contain host")
		assert.Contains(t, url, ":5433", "Should contain port")
		assert.Contains(t, url, "production", "Should contain database name")
		assert.Contains(t, url, "sslmode=require", "Should contain SSL mode")
	})
}

// BenchmarkDatabaseConfig_URL benchmarks URL generation performance
func BenchmarkDatabaseConfig_URL(b *testing.B) {
	config := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "benchuser",
		Password: "benchpass",
		DBName:   "benchdb",
		SSLMode:  "disable",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.URL()
	}
}

// BenchmarkDatabaseConfig_DSN benchmarks DSN generation performance
func BenchmarkDatabaseConfig_DSN(b *testing.B) {
	config := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "benchuser",
		Password: "benchpass",
		DBName:   "benchdb",
		SSLMode:  "disable",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.DSN()
	}
}

// TestConfig_AllFields tests that all config fields are properly loaded
func TestConfig_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "full.yaml")

	fullContent := `
app:
  name: "full-app"

server:
  host: "0.0.0.0"
  port: 9000

auth:
  keys_path: "/path/to/keys"
  active_kid: "production"

database:
  host: "db.example.com"
  port: 5433
  user: "dbuser"
  password: "dbpass123"
  dbname: "appdb"
  sslmode: "require"

logging:
  level: "debug"
`
	err := os.WriteFile(configPath, []byte(fullContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify each section
	assert.Equal(t, "full-app", cfg.App.Name)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 9000, cfg.Server.Port)
	assert.Equal(t, "/path/to/keys", cfg.Auth.KeysPath)
	assert.Equal(t, "production", cfg.Auth.ActiveKID)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "dbuser", cfg.Database.User)
	assert.Equal(t, "dbpass123", cfg.Database.Password)
	assert.Equal(t, "appdb", cfg.Database.DBName)
	assert.Equal(t, "require", cfg.Database.SSLMode)
	assert.Equal(t, "debug", cfg.Logging.Level)

	// Test helper methods
	assert.Equal(t, "0.0.0.0:9000", cfg.Server.Address())
	assert.Contains(t, cfg.Database.DSN(), "host=db.example.com")
	assert.Contains(t, cfg.Database.URL(), "postgres://")
}
