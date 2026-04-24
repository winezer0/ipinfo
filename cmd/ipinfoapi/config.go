package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ServerConfig server configuration structure
type ServerConfig struct {
	Auth     AuthConfig     `yaml:"auth"`
	HTTP     HTTPConfig     `yaml:"http"`
	Database DatabaseConfig `yaml:"database"`
	Log      LogConfig      `yaml:"log"`
}

// DatabaseConfig database configuration structure
type DatabaseConfig struct {
	AsnIpvxDb    string `yaml:"asn_ipv_x_db"`
	AsnIpv4Db    string `yaml:"asn_ipv4_db"`
	AsnIpv6Db    string `yaml:"asn_ipv6_db"`
	IpvxLocateDb string `yaml:"ipv_x_locate_db"`
	Ipv4LocateDb string `yaml:"ipv4_locate_db"`
	Ipv6LocateDb string `yaml:"ipv6_locate_db"`
}

// AuthConfig authentication configuration structure
type AuthConfig struct {
	Token  string `yaml:"token"`
	Enable bool   `yaml:"enable"`
}

// HTTPConfig HTTP service configuration structure
type HTTPConfig struct {
	Enable       bool   `yaml:"enable"`
	Port         int    `yaml:"port"`
	HTTPS        bool   `yaml:"https"`
	CertFile     string `yaml:"cert_file"`
	KeyFile      string `yaml:"key_file"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
	IdleTimeout  int    `yaml:"idle_timeout"`
}

// LogConfig logging configuration structure
type LogConfig struct {
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	Console    string `yaml:"console"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
}

// LoadConfig loads configuration from yaml file
func LoadConfig(appName string) (*ServerConfig, error) {
	configPath := findConfigFile(appName)
	if configPath == "" {
		return nil, fmt.Errorf("config file not found: %s.yaml", appName)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ServerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default values
	setDefaultConfig(&config)

	return &config, nil
}

// setDefaultConfig sets default configuration values
func setDefaultConfig(config *ServerConfig) {
	if config.HTTP.ReadTimeout == 0 {
		config.HTTP.ReadTimeout = 10
	}
	if config.HTTP.WriteTimeout == 0 {
		config.HTTP.WriteTimeout = 10
	}
	if config.HTTP.IdleTimeout == 0 {
		config.HTTP.IdleTimeout = 30
	}
	if config.Log.Level == "" {
		config.Log.Level = "info"
	}
	if config.Log.Console == "" {
		config.Log.Console = "TLCM"
	}
	if config.Log.MaxSize == 0 {
		config.Log.MaxSize = 100
	}
	if config.Log.MaxBackups == 0 {
		config.Log.MaxBackups = 3
	}
}

// findConfigFile finds configuration file
func findConfigFile(appName string) string {
	configFileName := fmt.Sprintf("%s.yaml", appName)

	// Current working directory
	if path := checkConfigFile(configFileName); path != "" {
		return path
	}

	// Executable directory
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		if path := checkConfigFile(filepath.Join(execDir, configFileName)); path != "" {
			return path
		}

		// Parent directory of executable
		parentDir := filepath.Dir(execDir)
		if path := checkConfigFile(filepath.Join(parentDir, configFileName)); path != "" {
			return path
		}
	}

	return ""
}

// checkConfigFile checks if file exists
func checkConfigFile(path string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}
