package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfig 测试配置文件加载
func TestLoadConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "iplocate.yaml")
	configContent := `
asn_ipvx_db: "assets/test-asn.mmdb"
asn_ipv4_db: "assets/test-asn-v4.mmdb"
asn_ipv6_db: "assets/test-asn-v6.mmdb"
ipvx_locate_db: "assets/test-locate.mmdb"
ipv4_locate_db: "assets/test-v4.db"
ipv6_locate_db: "assets/test-v6.db"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}

	config, err := LoadConfig("iplocate")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if config.AsnIpvxDb != "assets/test-asn.mmdb" {
		t.Errorf("AsnIpvxDb 期望 'assets/test-asn.mmdb', 得到 '%s'", config.AsnIpvxDb)
	}
	if config.AsnIpv4Db != "assets/test-asn-v4.mmdb" {
		t.Errorf("AsnIpv4Db 期望 'assets/test-asn-v4.mmdb', 得到 '%s'", config.AsnIpv4Db)
	}
	if config.AsnIpv6Db != "assets/test-asn-v6.mmdb" {
		t.Errorf("AsnIpv6Db 期望 'assets/test-asn-v6.mmdb', 得到 '%s'", config.AsnIpv6Db)
	}
	if config.IpvxLocateDb != "assets/test-locate.mmdb" {
		t.Errorf("IpvxLocateDb 期望 'assets/test-locate.mmdb', 得到 '%s'", config.IpvxLocateDb)
	}
	if config.Ipv4LocateDb != "assets/test-v4.db" {
		t.Errorf("Ipv4LocateDb 期望 'assets/test-v4.db', 得到 '%s'", config.Ipv4LocateDb)
	}
	if config.Ipv6LocateDb != "assets/test-v6.db" {
		t.Errorf("Ipv6LocateDb 期望 'assets/test-v6.db', 得到 '%s'", config.Ipv6LocateDb)
	}
}

// TestLoadConfig_Empty 测试空配置文件
func TestLoadConfig_Empty(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "iplocate.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}

	config, err := LoadConfig("iplocate")
	if err != nil {
		t.Fatalf("加载空配置失败: %v", err)
	}

	if config.AsnIpvxDb != "" {
		t.Errorf("空配置下 AsnIpvxDb 应为空, 得到 '%s'", config.AsnIpvxDb)
	}
	if config.IpvxLocateDb != "" {
		t.Errorf("空配置下 IpvxLocateDb 应为空, 得到 '%s'", config.IpvxLocateDb)
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

// TestLoadConfig_InvalidYAML 测试无效YAML格式
func TestLoadConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "iplocate.yaml")
	invalidContent := `
asn_ipvx_db: [invalid yaml
  - this is not valid
`
	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}

	_, err := LoadConfig("iplocate")
	if err == nil {
		t.Error("无效YAML格式应返回错误")
	}
}

// TestLoadConfigFromFile 测试从指定路径加载配置
func TestLoadConfigFromFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "custom.yaml")
	configContent := `
ipv4_locate_db: "assets/qqwry.dat"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	config, err := LoadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if config.Ipv4LocateDb != "assets/qqwry.dat" {
		t.Errorf("Ipv4LocateDb 期望 'assets/qqwry.dat', 得到 '%s'", config.Ipv4LocateDb)
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
