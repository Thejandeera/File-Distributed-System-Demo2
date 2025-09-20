package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

// Config holds the application configuration
type Config struct {
	Peers           []string `json:"peers"`
	StoragePath     string   `json:"storagePath"`
	QuotaLimit      int64    `json:"quotaLimit"`
	HeartbeatInterval int    `json:"heartbeatInterval"`
	RecoveryInterval int     `json:"recoveryInterval"`
	MaxFileSize     int64    `json:"maxFileSize"`
	EnableLogging   bool     `json:"enableLogging"`
	LogLevel        string   `json:"logLevel"`
}

// Default configuration
var defaultConfig = Config{
	Peers: []string{
		"http://localhost:8001",
		"http://localhost:8002",
	},
	StoragePath:      "./storage_data",
	QuotaLimit:       100 * 1024 * 1024, // 100 MB
	HeartbeatInterval: 5,                 // seconds
	RecoveryInterval: 30,                 // seconds
	MaxFileSize:      10 * 1024 * 1024,  // 10 MB
	EnableLogging:    true,
	LogLevel:         "INFO",
}

// Global configuration instance
var (
	globalConfig *Config
	configMu     sync.RWMutex
)

// GetConfig returns the global configuration
func GetConfig() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return globalConfig
}

// SetConfig sets the global configuration
func SetConfig(config *Config) {
	configMu.Lock()
	defer configMu.Unlock()
	globalConfig = config
}

// InitializeConfig initializes the configuration
func InitializeConfig() {
	config := &defaultConfig
	
	// Try to load from config file
	if err := loadConfigFromFile("config.json", config); err != nil {
		log.Printf("⚠️ Could not load config file, using defaults: %v", err)
	}
	
	SetConfig(config)
	log.Println("✅ Configuration initialized")
}

// loadConfigFromFile loads configuration from a JSON file
func loadConfigFromFile(filename string, config *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(config)
}

// SaveConfigToFile saves the current configuration to a JSON file
func SaveConfigToFile(filename string) error {
	config := GetConfig()
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}

// GetPeers returns the list of peer nodes
func GetPeers() []string {
	config := GetConfig()
	return config.Peers
}

// GetStoragePath returns the storage path
func GetStoragePath() string {
	config := GetConfig()
	return config.StoragePath
}

// GetQuotaLimit returns the quota limit
func GetQuotaLimit() int64 {
	config := GetConfig()
	return config.QuotaLimit
}

// GetHeartbeatInterval returns the heartbeat interval
func GetHeartbeatInterval() int {
	config := GetConfig()
	return config.HeartbeatInterval
}

// GetRecoveryInterval returns the recovery interval
func GetRecoveryInterval() int {
	config := GetConfig()
	return config.RecoveryInterval
}

// GetMaxFileSize returns the maximum file size
func GetMaxFileSize() int64 {
	config := GetConfig()
	return config.MaxFileSize
}

// IsLoggingEnabled returns whether logging is enabled
func IsLoggingEnabled() bool {
	config := GetConfig()
	return config.EnableLogging
}

// GetLogLevel returns the log level
func GetLogLevel() string {
	config := GetConfig()
	return config.LogLevel
}

// UpdateConfig updates the configuration
func UpdateConfig(updates map[string]interface{}) error {
	config := GetConfig()
	
	// Create a new config with updates
	newConfig := *config
	
	for key, value := range updates {
		switch key {
		case "peers":
			if peers, ok := value.([]string); ok {
				newConfig.Peers = peers
			}
		case "storagePath":
			if path, ok := value.(string); ok {
				newConfig.StoragePath = path
			}
		case "quotaLimit":
			if limit, ok := value.(int64); ok {
				newConfig.QuotaLimit = limit
			}
		case "heartbeatInterval":
			if interval, ok := value.(int); ok {
				newConfig.HeartbeatInterval = interval
			}
		case "recoveryInterval":
			if interval, ok := value.(int); ok {
				newConfig.RecoveryInterval = interval
			}
		case "maxFileSize":
			if size, ok := value.(int64); ok {
				newConfig.MaxFileSize = size
			}
		case "enableLogging":
			if enabled, ok := value.(bool); ok {
				newConfig.EnableLogging = enabled
			}
		case "logLevel":
			if level, ok := value.(string); ok {
				newConfig.LogLevel = level
			}
		}
	}
	
	SetConfig(&newConfig)
	return nil
}

// ValidateConfig validates the configuration
func ValidateConfig(config *Config) error {
	if config.QuotaLimit <= 0 {
		return fmt.Errorf("quota limit must be positive")
	}
	
	if config.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive")
	}
	
	if config.HeartbeatInterval <= 0 {
		return fmt.Errorf("heartbeat interval must be positive")
	}
	
	if config.RecoveryInterval <= 0 {
		return fmt.Errorf("recovery interval must be positive")
	}
	
	if config.StoragePath == "" {
		return fmt.Errorf("storage path cannot be empty")
	}
	
	return nil
}
