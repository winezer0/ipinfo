package zxwry

import (
	"path/filepath"
	"runtime"
	"testing"
)

func getTestDBPath(dbname string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..", "..")
	return filepath.Join(projectRoot, "assets", dbname)
}
func TestIpv6Location_FindFull(t *testing.T) {
	// 集成测试：测试完整的查询流程
	dbpath := getTestDBPath("zxipv6wry.db")
	db, err := NewZXWryDB(dbpath)
	if err != nil {
		t.Skipf("跳过测试，因为无法加载数据库: %v", err)
	}
	defer db.Close()

	// 测试一些常见的IPv6地址
	testIPs := []string{
		"2001:db8::1",
		"2405:6f00:c602::1",
		"2409:8c1e:75b0:1120::27",
		"2402:3c00:1000:4::1",
		"2408:8652:200::c101",
		"2409:8900:103f:14f:d7e:cd36:11af:be83",
		"fe80::5c12:27dc:93a4:3426",
		"::1",
	}

	for _, ip := range testIPs {
		t.Run(ip, func(t *testing.T) {
			result := db.FindFull(ip)

			// 记录结果用于调试
			t.Logf("查询IP: %s -> 位置: %s, 结构化: Country=%s, Province=%s, City=%s, ISP=%s",
				ip, result.Location, result.Country, result.Province, result.City, result.ISP)
		})
	}
}

func TestIpv6Location_BatchFindFull(t *testing.T) {
	dbpath := getTestDBPath("zxipv6wry.db")
	db, err := NewZXWryDB(dbpath)
	if err != nil {
		t.Skipf("跳过测试，因为无法加载数据库: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"2001:db8::1",
		"2405:6f00:c602::1",
		"2409:8c1e:75b0:1120::27",
		"invalid_ip",
		"192.168.1.1",
	}

	results := db.BatchFindFull(testIPs)

	if len(results) != len(testIPs) {
		t.Errorf("期望结果数量为 %d，但得到了 %d", len(testIPs), len(results))
	}

	for ip, result := range results {
		t.Logf("批量查询 - IP: %s -> 位置: %s, Country=%s", ip, result.Location, result.Country)
	}
}

func TestIpv6Location_GetDatabaseInfo(t *testing.T) {
	dbpath := getTestDBPath("zxipv6wry.db")
	db, err := NewZXWryDB(dbpath)
	if err != nil {
		t.Skipf("跳过测试，因为无法加载数据库: %v", err)
	}
	defer db.Close()

	info := db.GetDatabaseInfo()

	if info.Type != "zxwry" {
		t.Errorf("数据库类型错误: %v, 期望 zxwry", info.Type)
	}

	if info.IsIPv4 {
		t.Error("数据库不应该支持 IPv4")
	}

	if !info.IsIPv6 {
		t.Error("数据库应该支持 IPv6")
	}

	t.Logf("数据库信息: %+v", info)
}

// TestZXWryFindAndFindFull 测试Find和FindFull方法对比
func TestZXWryFindAndFindFull(t *testing.T) {
	dbPath := getTestDBPath("zxipv6wry.db")

	db, err := NewZXWryDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"2001:db8::1",
		"2405:6f00:c602::1",
		"2409:8c1e:75b0:1120::27",
		"2402:3c00:1000:4::1",
		"2408:8652:200::c101",
		"2409:8900:103f:14f:d7e:cd36:11af:be83",
		"::1",
	}

	for _, ip := range testIPs {
		findFullResult := db.FindFull(ip)

		t.Logf("=== IP: %s ===", ip)
		t.Logf("FindFull 结果: Location=%s, Country=%s, Province=%s, City=%s, Area=%s, ISP=%s",
			findFullResult.Location, findFullResult.Country, findFullResult.Province, findFullResult.City, findFullResult.Area, findFullResult.ISP)
	}
}
