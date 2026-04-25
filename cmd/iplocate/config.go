package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AppConfig 应用程序配置结构体
type AppConfig struct {
	// ASN数据库配置
	AsnIpvxDb string `yaml:"asn_ipvx_db"`
	AsnIpv4Db string `yaml:"asn_ipv4_db"`
	AsnIpv6Db string `yaml:"asn_ipv6_db"`

	// IP地理位置数据库配置
	IpvxLocateDb string `yaml:"ipvx_locate_db"`
	Ipv4LocateDb string `yaml:"ipv4_locate_db"`
	Ipv6LocateDb string `yaml:"ipv6_locate_db"`
}

// LoadConfig 从yaml文件加载配置
// 默认配置文件名为 <appName>.yaml
func LoadConfig(appName string) (*AppConfig, error) {
	configPath := findConfigFile(appName)
	if configPath == "" {
		return nil, fmt.Errorf("未找到配置文件 %s.yaml", appName)
	}
	return LoadConfigFromFile(configPath)
}

// LoadConfigFromFile 从指定路径加载配置文件
func LoadConfigFromFile(configPath string) (*AppConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &config, nil
}

// findConfigFile 查找配置文件
// 搜索顺序:
// 1. 当前工作目录
// 2. 可执行文件所在目录
// 3. 可执行文件所在目录的上级目录
func findConfigFile(appName string) string {
	configFileName := fmt.Sprintf("%s.yaml", appName)

	// 1. 当前工作目录
	if path := checkConfigFile(configFileName); path != "" {
		return path
	}

	// 2. 可执行文件所在目录
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		if path := checkConfigFile(filepath.Join(execDir, configFileName)); path != "" {
			return path
		}

		// 3. 可执行文件所在目录的上级目录
		parentDir := filepath.Dir(execDir)
		if path := checkConfigFile(filepath.Join(parentDir, configFileName)); path != "" {
			return path
		}
	}

	return ""
}

// checkConfigFile 检查文件是否存在
func checkConfigFile(path string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}
