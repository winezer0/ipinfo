package ipdb

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func getTestDBPath(dbname string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..", "..")
	return filepath.Join(projectRoot, "assets", dbname)
}

func TestNewIPDB_RealDB(t *testing.T) {
	dbPath := getTestDBPath("qqwry.ipdb")

	db, err := NewIPDB(dbPath)
	if err != nil {
		t.Fatalf("NewIPDB() error = %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("NewIPDB() returned nil")
	}

	if db.city == nil {
		t.Fatal("NewIPDB() city is nil")
	}
}

func TestIPDB_DatabaseInfo(t *testing.T) {
	dbPath := getTestDBPath("qqwry.ipdb")

	db, err := NewIPDB(dbPath)
	if err != nil {
		t.Fatalf("NewIPDB() error = %v", err)
	}
	defer db.Close()

	info := db.GetDatabaseInfo()

	if info.Type != "ipdb" {
		t.Errorf("GetDatabaseInfo() type = %v, want ipdb", info.Type)
	}

	if !info.IsIPv4 {
		t.Error("GetDatabaseInfo() is_ipv4 should be true")
	}

	if !info.IsIPv6 {
		t.Error("GetDatabaseInfo() is_ipv6 should be true")
	}

	t.Logf("数据库信息: %+v", info)
}

func TestIPDB_IPv4_Query(t *testing.T) {
	dbPath := getTestDBPath("qqwry.ipdb")

	db, err := NewIPDB(dbPath)
	if err != nil {
		t.Fatalf("NewIPDB() error = %v", err)
	}
	defer db.Close()

	tests := []struct {
		name      string
		ip        string
		wantEmpty bool
	}{
		{
			name:      "Google DNS",
			ip:        "8.8.8.8",
			wantEmpty: false,
		},
		{
			name:      "Cloudflare DNS",
			ip:        "1.1.1.1",
			wantEmpty: false,
		},
		{
			name:      "114 DNS",
			ip:        "114.114.114.114",
			wantEmpty: false,
		},
		{
			name:      "百度",
			ip:        "220.181.38.148",
			wantEmpty: false,
		},
		{
			name:      "淘宝",
			ip:        "140.205.220.96",
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := db.FindFull(tt.ip)
			if tt.wantEmpty && result.Location != "" {
				t.Errorf("FindFull(%s) = %v, want empty", tt.ip, result)
			}
			if !tt.wantEmpty && result.Location == "" {
				t.Errorf("FindFull(%s) returned empty, want non-empty", tt.ip)
			}
			t.Logf("IP: %s -> %s", tt.ip, result.Location)
		})
	}
}

func TestIPDB_IPv6_Query(t *testing.T) {
	dbPath := getTestDBPath("qqwry.ipdb")

	db, err := NewIPDB(dbPath)
	if err != nil {
		t.Fatalf("NewIPDB() error = %v", err)
	}
	defer db.Close()

	tests := []struct {
		name      string
		ip        string
		wantEmpty bool
	}{
		{
			name:      "阿里云 IPv6",
			ip:        "2400:3200::1",
			wantEmpty: false,
		},
		{
			name:      "Google IPv6",
			ip:        "2001:4860:4860::8888",
			wantEmpty: false,
		},
		{
			name:      "腾讯云 IPv6",
			ip:        "2402:4e00::",
			wantEmpty: false,
		},
		{
			name:      "IPv6 本地回环",
			ip:        "::1",
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := db.FindFull(tt.ip)
			if tt.wantEmpty && result.Location != "" {
				t.Errorf("FindFull(%s) = %v, want empty", tt.ip, result)
			}
			if !tt.wantEmpty && result.Location == "" {
				t.Errorf("FindFull(%s) returned empty, want non-empty", tt.ip)
			}
			t.Logf("IP: %s -> %s", tt.ip, result.Location)
		})
	}
}

func TestIPDB_BatchFindFull(t *testing.T) {
	dbPath := getTestDBPath("qqwry.ipdb")

	db, err := NewIPDB(dbPath)
	if err != nil {
		t.Fatalf("NewIPDB() error = %v", err)
	}
	defer db.Close()

	queries := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
		"2400:3200::1",
		"2001:4860:4860::8888",
	}

	results := db.BatchFindFull(queries)

	if len(results) != len(queries) {
		t.Errorf("BatchFindFull() returned %d results, want %d", len(results), len(queries))
	}

	for _, query := range queries {
		result, exists := results[query]
		if !exists {
			t.Errorf("BatchFindFull() missing result for %s", query)
		}
		t.Logf("IP: %s -> %s", query, result.Location)
	}
}

func TestIPDB_IPv4_IPv6_Support(t *testing.T) {
	dbPath := getTestDBPath("qqwry.ipdb")

	db, err := NewIPDB(dbPath)
	if err != nil {
		t.Fatalf("NewIPDB() error = %v", err)
	}
	defer db.Close()

	info := db.GetDatabaseInfo()

	isIPv4 := info.IsIPv4
	isIPv6 := info.IsIPv6

	if !isIPv4 {
		t.Error("数据库应该支持 IPv4")
	}

	if !isIPv6 {
		t.Error("数据库应该支持 IPv6")
	}

	ipv4Result := db.FindFull("8.8.8.8")
	if ipv4Result.Location == "" {
		t.Error("IPv4 查询失败")
	} else {
		t.Logf("IPv4 查询成功: 8.8.8.8 -> %s", ipv4Result.Location)
	}

	ipv6Result := db.FindFull("2400:3200::1")
	if ipv6Result.Location == "" {
		t.Error("IPv6 查询失败")
	} else {
		t.Logf("IPv6 查询成功: 2400:3200::1 -> %s", ipv6Result.Location)
	}
}

func TestIPDB_InvalidIP(t *testing.T) {
	dbPath := getTestDBPath("qqwry.ipdb")

	db, err := NewIPDB(dbPath)
	if err != nil {
		t.Fatalf("NewIPDB() error = %v", err)
	}
	defer db.Close()

	result := db.FindFull("invalid_ip")
	if result.Location != "" {
		t.Errorf("FindFull(invalid_ip) = %v, want empty", result)
	}
}

func TestIPDB_ChineseLocation(t *testing.T) {
	dbPath := getTestDBPath("qqwry.ipdb")

	db, err := NewIPDB(dbPath)
	if err != nil {
		t.Fatalf("NewIPDB() error = %v", err)
	}
	defer db.Close()

	tests := []struct {
		name string
		ip   string
	}{
		{"北京", "202.108.22.5"},
		{"上海", "202.96.209.5"},
		{"广州", "202.96.128.68"},
		{"深圳", "202.96.134.134"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := db.FindFull(tt.ip)
			if result.Location == "" {
				t.Errorf("FindFull(%s) returned empty", tt.ip)
			}
			if !strings.Contains(result.Location, "中国") {
				t.Errorf("FindFull(%s) = %v, should contain 中国", tt.ip, result.Location)
			}
			t.Logf("IP: %s -> %s", tt.ip, result.Location)
		})
	}
}

// TestIPDBFindAndFindFull 测试Find和FindFull方法对比
func TestIPDBFindAndFindFull(t *testing.T) {
	dbPath := getTestDBPath("qqwry.ipdb")

	db, err := NewIPDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
		"220.181.38.148",
		"202.108.22.5",
	}

	for _, ip := range testIPs {
		findFullResult := db.FindFull(ip)

		t.Logf("=== IP: %s ===", ip)
		t.Logf("FindFull 结果: Location=%s, Country=%s, Province=%s, City=%s, Area=%s, ISP=%s",
			findFullResult.Location, findFullResult.Country, findFullResult.Province, findFullResult.City, findFullResult.Area, findFullResult.ISP)
	}
}

// TestIPDBIPv6FindAndFindFull 测试IPv6的FindFull方法
func TestIPDBIPv6FindAndFindFull(t *testing.T) {
	dbPath := getTestDBPath("qqwry.ipdb")

	db, err := NewIPDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"2400:3200::1",
		"2001:4860:4860::8888",
		"2402:4e00::",
		"::1",
	}

	for _, ip := range testIPs {
		findFullResult := db.FindFull(ip)

		t.Logf("=== IP: %s ===", ip)
		t.Logf("FindFull 结果: Location=%s, Country=%s, Province=%s, City=%s, Area=%s, ISP=%s",
			findFullResult.Location, findFullResult.Country, findFullResult.Province, findFullResult.City, findFullResult.Area, findFullResult.ISP)
	}
}
