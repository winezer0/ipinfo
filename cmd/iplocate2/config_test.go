package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfig 测试配置文件加载
func TestLoadConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "iplocate2.yaml")
	configContent := `
ip_locate_dbs:
  - "assets/qqwry.dat"
  - "assets/ip2region.xdb"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}

	config, err := LoadConfig("iplocate2")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if len(config.IpLocateDbs) != 2 {
		t.Errorf("IpLocateDbs 期望 2 个, 得到 %d 个", len(config.IpLocateDbs))
	}
	if config.IpLocateDbs[0] != "assets/qqwry.dat" {
		t.Errorf("IpLocateDbs[0] 期望 'assets/qqwry.dat', 得到 '%s'", config.IpLocateDbs[0])
	}
	if config.IpLocateDbs[1] != "assets/ip2region.xdb" {
		t.Errorf("IpLocateDbs[1] 期望 'assets/ip2region.xdb', 得到 '%s'", config.IpLocateDbs[1])
	}
}

// TestLoadConfig_Empty 测试空配置文件
func TestLoadConfig_Empty(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "iplocate2.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}

	config, err := LoadConfig("iplocate2")
	if err != nil {
		t.Fatalf("加载空配置失败: %v", err)
	}

	if len(config.IpLocateDbs) != 0 {
		t.Errorf("空配置下 IpLocateDbs 应为空, 得到 %d 个", len(config.IpLocateDbs))
	}
}

// TestLoadConfig_NotFound 测试配置文件不存在
func TestLoadConfig_NotFound(t *testing.T) {
	tempDir := t.TempDir()

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}

	_, err := LoadConfig("nonexistent")
	if err == nil {
		t.Error("配置文件不存在时应返回错误")
	}
}

// TestLoadConfigFromFile 测试从指定路径加载配置
func TestLoadConfigFromFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "custom.yaml")
	configContent := `
ip_locate_dbs:
  - "assets/qqwry.dat"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	config, err := LoadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if len(config.IpLocateDbs) != 1 {
		t.Errorf("IpLocateDbs 期望 1 个, 得到 %d 个", len(config.IpLocateDbs))
	}
	if config.IpLocateDbs[0] != "assets/qqwry.dat" {
		t.Errorf("IpLocateDbs[0] 期望 'assets/qqwry.dat', 得到 '%s'", config.IpLocateDbs[0])
	}
}

// TestLoadConfigFromFile_NotFound 测试配置文件不存在
func TestLoadConfigFromFile_NotFound(t *testing.T) {
	_, err := LoadConfigFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("配置文件不存在时应返回错误")
	}
}

// TestFindConfigFile 测试配置文件查找
func TestFindConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "testapp.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}

	found := findConfigFile("testapp")
	if found == "" {
		t.Error("应找到配置文件")
	}

	found = findConfigFile("nonexistent")
	if found != "" {
		t.Error("不应找到不存在的配置文件")
	}
}
