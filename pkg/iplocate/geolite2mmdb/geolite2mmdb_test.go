// Package geolite2mmdb 提供对 GeoLite2 格式 MMDB 数据库的支持
package geolite2mmdb

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

// TestGeoLite2MMDBIPv4 测试GeoLite2 IPv4数据库查询
func TestGeoLite2MMDBIPv4(t *testing.T) {
	dbPath := getTestDBPath("geolite2-city-ipv4.mmdb")

	db, err := NewGeoLite2MMDB(dbPath)
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

// TestGeoLite2MMDBIPv6 测试GeoLite2 IPv6数据库查询
func TestGeoLite2MMDBIPv6(t *testing.T) {
	dbPath := getTestDBPath("geolite2-city-ipv6.mmdb")

	db, err := NewGeoLite2MMDB(dbPath)
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

// TestGeoLite2MMDBDualStack 测试GeoLite2双栈数据库查询
func TestGeoLite2MMDBDualStack(t *testing.T) {
	dbPath := getTestDBPath("geolite2-asn.mmdb")

	db, err := NewGeoLite2MMDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
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

// TestGeoLite2MMDBBatchFindFull 测试批量查询
func TestGeoLite2MMDBBatchFindFull(t *testing.T) {
	dbPath := getTestDBPath("geolite2-asn.mmdb")

	db, err := NewGeoLite2MMDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"2001:4860:4860::8888",
	}

	results := db.BatchFindFull(testIPs)
	if len(results) != len(testIPs) {
		t.Errorf("期望结果数量为 %d，但得到了 %d", len(testIPs), len(results))
	}

	for ip, result := range results {
		t.Logf("批量查询IP %s 结果: %s", ip, result.Location)
	}
}

// TestGeoLite2MMDBInvalidIP 测试无效IP查询
func TestGeoLite2MMDBInvalidIP(t *testing.T) {
	dbPath := getTestDBPath("geolite2-asn.mmdb")

	db, err := NewGeoLite2MMDB(dbPath)
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

// TestGeoLite2MMDBEmptyPath 测试空路径初始化
func TestGeoLite2MMDBEmptyPath(t *testing.T) {
	_, err := NewGeoLite2MMDB("")
	if err == nil {
		t.Error("期望初始化空路径失败，但成功了")
	}
}

// TestGeoLite2MMDBNonExistentFile 测试不存在的文件
func TestGeoLite2MMDBNonExistentFile(t *testing.T) {
	_, err := NewGeoLite2MMDB("/nonexistent/path/to/db.mmdb")
	if err == nil {
		t.Error("期望初始化不存在的文件失败，但成功了")
	}
}

// TestGeoLite2MMDBDatabaseType 测试数据库类型常量
func TestGeoLite2MMDBDatabaseType(t *testing.T) {
	dbPath := getTestDBPath("geolite2-asn.mmdb")

	db, err := NewGeoLite2MMDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	dbInfo := db.GetDatabaseInfo()
	if dbInfo.Type != iplocate.DBTypeGeoLiteMMDB {
		t.Errorf("期望数据库类型为 %s，但得到了 %s", iplocate.DBTypeGeoLiteMMDB, dbInfo.Type)
	}
}

// TestGeoLite2MMDBFindAndFindFull 测试Find和FindFull方法对比
func TestGeoLite2MMDBFindAndFindFull(t *testing.T) {
	dbPath := getTestDBPath("geolite2-city-ipv4.mmdb")

	db, err := NewGeoLite2MMDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
		"223.5.5.5",
	}

	for _, ip := range testIPs {
		findFullResult := db.FindFull(ip)

		t.Logf("=== IP: %s ===", ip)
		t.Logf("FindFull 结果: Location=%s, Country=%s, Province=%s, City=%s, Area=%s, ISP=%s",
			findFullResult.Location, findFullResult.Country, findFullResult.Province, findFullResult.City, findFullResult.Area, findFullResult.ISP)
	}
}

// TestGeoLite2MMDBIPv6FindAndFindFull 测试IPv6的FindFull方法
func TestGeoLite2MMDBIPv6FindAndFindFull(t *testing.T) {
	dbPath := getTestDBPath("geolite2-city-ipv6.mmdb")

	db, err := NewGeoLite2MMDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"2001:4860:4860::8888",
		"2606:4700::6813",
		"2400:3200::1",
	}

	for _, ip := range testIPs {
		findFullResult := db.FindFull(ip)

		t.Logf("=== IP: %s ===", ip)
		t.Logf("FindFull 结果: Location=%s, Country=%s, Province=%s, City=%s, Area=%s, ISP=%s",
			findFullResult.Location, findFullResult.Country, findFullResult.Province, findFullResult.City, findFullResult.Area, findFullResult.ISP)
	}
}
