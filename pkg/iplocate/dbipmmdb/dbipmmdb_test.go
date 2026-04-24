// Package dbipmmdb 提供对 DBIP 格式 MMDB 数据库的支持
package dbipmmdb

import (
	"github.com/winezer0/ipinfo/pkg/iplocate"
	"path/filepath"
	"runtime"
	"testing"
)

func getTestDBPath(dbname string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..", "..")
	return filepath.Join(projectRoot, "assets", dbname)
}

// TestDBIPMMDBIPv4 测试DBIP IPv4数据库查询
func TestDBIPMMDBIPv4(t *testing.T) {
	dbPath := getTestDBPath("dbip-city-ipv4.mmdb")

	db, err := NewDBIPMMDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
	}

	for _, ip := range testIPs {
		result := db.FindFull(ip)
		t.Logf("IP %s 查询结果: %s", ip, result.Location)
	}

	dbInfo := db.GetDatabaseInfo()
	if dbInfo == nil {
		t.Error("获取数据库信息失败")
	} else {
		t.Logf("数据库信息: %+v", dbInfo)
	}
}

// TestDBIPMMDBIPv6 测试DBIP IPv6数据库查询
func TestDBIPMMDBIPv6(t *testing.T) {
	dbPath := getTestDBPath("dbip-city-ipv6.mmdb")

	db, err := NewDBIPMMDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"2001:4860:4860::8888",
		"2606:4700::6813",
	}

	for _, ip := range testIPs {
		result := db.FindFull(ip)
		t.Logf("IP %s 查询结果: %s", ip, result.Location)
	}

	dbInfo := db.GetDatabaseInfo()
	if dbInfo == nil {
		t.Error("获取数据库信息失败")
	} else {
		t.Logf("数据库信息: %+v", dbInfo)
	}
}

// TestDBIPMMDBDualStack 测试DBIP双栈数据库查询
func TestDBIPMMDBDualStack(t *testing.T) {
	dbPath := getTestDBPath("dbip-city-ipv4.mmdb")

	db, err := NewDBIPMMDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
	}

	for _, ip := range testIPs {
		result := db.FindFull(ip)
		t.Logf("IP %s 查询结果: %s", ip, result.Location)
	}

	dbInfo := db.GetDatabaseInfo()
	if dbInfo == nil {
		t.Error("获取数据库信息失败")
	} else {
		t.Logf("数据库信息: %+v", dbInfo)
	}
}

// TestDBIPMMDBBatchFindFull 测试批量查询
func TestDBIPMMDBBatchFindFull(t *testing.T) {
	dbPath := getTestDBPath("dbip-city-ipv4.mmdb")

	db, err := NewDBIPMMDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
	}

	results := db.BatchFindFull(testIPs)
	if len(results) != len(testIPs) {
		t.Errorf("期望结果数量为 %d，但得到了 %d", len(testIPs), len(results))
	}

	for ip, result := range results {
		t.Logf("批量查询IP %s 结果: %s", ip, result.Location)
	}
}

// TestDBIPMMDBInvalidIP 测试无效IP查询
func TestDBIPMMDBInvalidIP(t *testing.T) {
	dbPath := getTestDBPath("dbip-city-ipv4.mmdb")

	db, err := NewDBIPMMDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	invalidIPs := []string{
		"invalid_ip",
		"",
		"999.999.999.999",
	}

	for _, ip := range invalidIPs {
		result := db.FindFull(ip)
		if result.Location != "" {
			t.Errorf("查询无效IP %s 应该返回空字符串，但得到了: %s", ip, result.Location)
		}
	}
}

// TestDBIPMMDBEmptyPath 测试空路径初始化
func TestDBIPMMDBEmptyPath(t *testing.T) {
	_, err := NewDBIPMMDB("")
	if err == nil {
		t.Error("期望初始化空路径失败，但成功了")
	}
}

// TestDBIPMMDBNonExistentFile 测试不存在的文件
func TestDBIPMMDBNonExistentFile(t *testing.T) {
	_, err := NewDBIPMMDB("/nonexistent/path/to/db.mmdb")
	if err == nil {
		t.Error("期望初始化不存在的文件失败，但成功了")
	}
}

// TestDBIPMMDBDatabaseType 测试数据库类型常量
func TestDBIPMMDBDatabaseType(t *testing.T) {
	dbPath := getTestDBPath("dbip-city-ipv4.mmdb")

	db, err := NewDBIPMMDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	dbInfo := db.GetDatabaseInfo()
	if dbInfo.Type != iplocate.DBTypeDBIPMMDB {
		t.Errorf("期望数据库类型为 %s，但得到了 %s", iplocate.DBTypeDBIPMMDB, dbInfo.Type)
	}
}

// TestDBIPMMDBFindFull 测试FindFull方法
func TestDBIPMMDBFindFull(t *testing.T) {
	dbPath := getTestDBPath("dbip-city-ipv4.mmdb")

	db, err := NewDBIPMMDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
	}

	for _, ip := range testIPs {
		result := db.FindFull(ip)
		if result == nil {
			t.Errorf("FindFull(%s) 返回 nil", ip)
			continue
		}

		t.Logf("IP: %s -> Country: %s, Province: %s, City: %s, Area: %s, ISP: %s",
			ip, result.Country, result.Province, result.City, result.Area, result.ISP)

		if result.IP != ip {
			t.Errorf("FindFull(%s).IP = %s, want %s", ip, result.IP, ip)
		}

		if result.Version != 4 {
			t.Errorf("FindFull(%s).Version = %d, want 4", ip, result.Version)
		}
	}
}
