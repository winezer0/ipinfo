package config

import (
	"fmt"
	"github.com/winezer0/ipinfo/cmd/ipinfoapi/embeds"
	"github.com/winezer0/ipinfo/pkg/iplocate/ip2region"
	"net"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadConfig 加载配置文件
// 当用户没有指定配置文件路径时, 先从当前目录和用户目录/.config查找 <AppName>.yaml,
// 找不到时使用内嵌模板；已找到的配置读取失败时返回错误。
func LoadConfig(cfgPath string, appName string) (*ServerConfig, error) {
	var data []byte
	var err error

	if cfgPath == "" {
		// 当用户没有指定配置文件路径时, 先从当前目录和用户目录/.config查找 <AppName>.yaml,
		// 仅找不到时使用内嵌配置，读取错误直接返回。
		defaultConfigPath := appName + ".yaml"
		cfgPath = findConfigPath(defaultConfigPath)
		if cfgPath != "" {
			data, err = os.ReadFile(cfgPath)
			if err != nil {
				return nil, fmt.Errorf("read found config %s: %w", cfgPath, err)
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
	if err := config.Validate(); err != nil {
		return nil, err
	}

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
	if config.XDBPoolSize == 0 {
		config.XDBPoolSize = 1
	}
	if config.HTTP.ReadTimeout == 0 {
		config.HTTP.ReadTimeout = 10
	}
	if config.HTTP.WriteTimeout == 0 {
		config.HTTP.WriteTimeout = 10
	}
	if config.HTTP.IdleTimeout == 0 {
		config.HTTP.IdleTimeout = 30
	}
	if config.HTTP.MaxBatchSize == 0 {
		config.HTTP.MaxBatchSize = 100
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
	return os.WriteFile(configPath, []byte(GetDefaultConfig()), 0644)
}

// Validate 校验请求限制和可信代理网段。
func (config *ServerConfig) Validate() error {
	if config.XDBPoolSize < 0 || config.XDBPoolSize > ip2region.MaxPoolSize {
		return fmt.Errorf("xdb_pool_size 必须为 1–%d（0 使用默认值1）", ip2region.MaxPoolSize)
	}
	if config.HTTP.MaxBatchSize < 1 {
		return fmt.Errorf("http.max_batch_size 必须大于0")
	}
	for _, proxy := range config.HTTP.TrustedProxies {
		if _, _, err := net.ParseCIDR(proxy); err != nil {
			return fmt.Errorf("无效的可信代理网段 %q: %w", proxy, err)
		}
	}
	return nil
}
