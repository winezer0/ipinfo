package config

import "github.com/winezer0/downutils/downutils"

// ServerConfig server configuration structure
type ServerConfig struct {
	Auth      AuthConfig           `yaml:"auth"`
	HTTP      HTTPConfig           `yaml:"http"`
	Log       LogConfig            `yaml:"log"`
	Databases []downutils.DownItem `yaml:"databases"`
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
