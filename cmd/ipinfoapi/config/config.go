package config

import (
	"fmt"
	"github.com/winezer0/ipinfo/cmd/ipinfoapi/embeds"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadConfig 加载配置文件
// 当用户没有指定配置文件路径时, 先从当前目录和用户目录/.config查找 <AppName>.yaml,
// 找不到时 或者加载错误时 使用内嵌的配置文件
func LoadConfig(cfgPath string, appName string) (*ServerConfig, error) {
	var data []byte
	var err error

	if cfgPath == "" {
		// 当用户没有指定配置文件路径时, 先从当前目录和用户目录/.config查找 <AppName>.yaml,
		// 找不到时 或者加载错误时 使用内嵌的配置文件
		defaultConfigPath := appName + ".yaml"
		cfgPath = findConfigPath(defaultConfigPath)
		if cfgPath != "" {
			data, err = os.ReadFile(cfgPath)
			if err != nil {
				fmt.Errorf("read found config %s error: %v", cfgPath, err)
				data = []byte(GetDefaultConfig())
			}
		} else {
			data = []byte(GetDefaultConfig())
		}
	} else {
		// 如果已指定配置文件 就从指定的配置中读取
		data, err = os.ReadFile(cfgPath)
		if err != nil {
			return nil, err
		}
	}

	var config ServerConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	setDefaultConfig(&config)

	return &config, nil
}

func findConfigPath(configName string) string {
	configPath := ""
	configPaths := []string{
		configName,
		filepath.Join(os.Getenv("HOME"), ".config", configName),
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}
	return configPath
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

// GetDefaultConfig 获取内置默认配置文件内容（从嵌入文件获取）
func GetDefaultConfig() string {
	return embeds.GetConfig()
}

// GenDefaultConfig 生成默认配置文件到指定路径
func GenDefaultConfig(configPath string) error {
	defaultConfig := GetDefaultConfig()
	return os.WriteFile(configPath, []byte(defaultConfig), 0644)
}
