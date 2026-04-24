package ip2region

import (
	"github.com/winezer0/ipinfo/pkg/iplocate"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

func getTestDBPath(dbname string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..", "..")
	return filepath.Join(projectRoot, "assets", dbname)
}

// TestIP2RegionIPv4 测试IPv4数据库查询
func TestIP2RegionIPv4(t *testing.T) {
	// xdb IPv4文件路径
	dbPath := getTestDBPath("ip2region_v4.xdb")

	// 创建IP2Region实例
	ip2Region, err := NewIP2Region(iplocate.IPv4VersionNo, dbPath)
	if err != nil {
		t.Fatalf("创建IPv4 IP2Region实例失败: %v", err)
	}
	defer ip2Region.Close()

	// 测试单个IP查询
	testIP := "8.8.8.8"
	result := ip2Region.FindFull(testIP)
	if result.Location == "" {
		t.Errorf("查询IP %s 失败，返回空结果", testIP)
	} else {
		t.Logf("查询IP %s 结果: %s", testIP, result.Location)
	}

	// 测试批量查询
	testIPs := []string{"114.114.114.114", "1.1.1.1", "8.8.4.4"}
	batchResults := ip2Region.BatchFindFull(testIPs)
	for ip, result := range batchResults {
		if result.Location == "" {
			t.Errorf("批量查询IP %s 失败，返回空结果", ip)
		} else {
			t.Logf("批量查询IP %s 结果: %s", ip, result.Location)
		}
	}

	// 测试获取数据库信息
	dbInfo := ip2Region.GetDatabaseInfo()
	if dbInfo == nil {
		t.Error("获取数据库信息失败")
	} else {
		t.Logf("数据库信息: %+v", dbInfo)
	}
}

// TestIP2RegionIPv6 测试IPv6数据库查询
func TestIP2RegionIPv6(t *testing.T) {
	// xdb IPv6文件路径
	dbPath := getTestDBPath("ip2region_v6.xdb")

	// 创建IP2Region实例
	ip2Region, err := NewIP2Region(iplocate.IPv6VersionNo, dbPath)
	if err != nil {
		t.Fatalf("创建IPv6 IP2Region实例失败: %v", err)
	}
	defer ip2Region.Close()

	// 测试单个IPv6查询
	testIP := "2001:4860:4860::8888"
	result := ip2Region.FindFull(testIP)
	if result.Location == "" {
		t.Errorf("查询IPv6 %s 失败，返回空结果", testIP)
	} else {
		t.Logf("查询IPv6 %s 结果: %s", testIP, result.Location)
	}

	// 测试批量IPv6查询
	testIPs := []string{"2001:4860:4860::8844", "2606:4700:4700::1111"}
	batchResults := ip2Region.BatchFindFull(testIPs)
	for ip, result := range batchResults {
		if result.Location == "" {
			t.Errorf("批量查询IPv6 %s 失败，返回空结果", ip)
		} else {
			t.Logf("批量查询IPv6 %s 结果: %s", ip, result.Location)
		}
	}

	// 测试获取数据库信息
	dbInfo := ip2Region.GetDatabaseInfo()
	if dbInfo == nil {
		t.Error("获取数据库信息失败")
	} else {
		t.Logf("数据库信息: %+v", dbInfo)
	}
}

// TestIP2RegionWithVectorIndex 测试使用 VectorIndex 缓存的查询
func TestIP2RegionWithVectorIndex(t *testing.T) {
	// xdb IPv4文件路径
	dbPath := getTestDBPath("ip2region_v4.xdb")

	// 预加载VectorIndex
	vectorIndex, err := LoadVectorIndexFromFile(dbPath)
	if err != nil {
		t.Fatalf("加载VectorIndex失败: %v", err)
	}

	// 创建使用VectorIndex的IP2Region实例
	ip2Region, err := NewIP2RegionWithVectorIndex(iplocate.IPv4VersionNo, dbPath, vectorIndex)
	if err != nil {
		t.Fatalf("创建使用VectorIndex的IP2Region实例失败: %v", err)
	}
	defer ip2Region.Close()

	// 测试查询
	testIP := "114.114.114.114"
	result := ip2Region.FindFull(testIP)
	if result.Location == "" {
		t.Errorf("使用VectorIndex查询IP %s 失败，返回空结果", testIP)
	} else {
		t.Logf("使用VectorIndex查询IP %s 结果: %s", testIP, result.Location)
	}
}

// TestIP2RegionVerify 测试xdb文件验证功能
func TestIP2RegionVerify(t *testing.T) {
	// 验证IPv4 xdb文件
	dbPath := getTestDBPath("ip2region_v4.xdb")
	err := xdb.VerifyFromFile(dbPath)
	if err != nil {
		t.Errorf("验证IPv4 xdb文件失败: %v", err)
	} else {
		t.Log("IPv4 xdb文件验证成功")
	}

	// 验证IPv6 xdb文件
	dbPath = getTestDBPath("ip2region_v6.xdb")
	err = xdb.VerifyFromFile(dbPath)
	if err != nil {
		t.Errorf("验证IPv6 xdb文件失败: %v", err)
	} else {
		t.Log("IPv6 xdb文件验证成功")
	}
}

// TestIP2RegionFindAndFindFull 测试Find和FindFull方法对比
func TestIP2RegionFindAndFindFull(t *testing.T) {
	dbPath := getTestDBPath("ip2region_v4.xdb")

	ip2Region, err := NewIP2Region(iplocate.IPv4VersionNo, dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer ip2Region.Close()

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
		"223.5.5.5",
	}

	for _, ip := range testIPs {
		findFullResult := ip2Region.FindFull(ip)

		t.Logf("=== IP: %s ===", ip)
		t.Logf("FindFull 结果: Location=%s, Country=%s, Province=%s, City=%s, Area=%s, ISP=%s",
			findFullResult.Location, findFullResult.Country, findFullResult.Province, findFullResult.City, findFullResult.Area, findFullResult.ISP)
	}
}

// TestIP2RegionIPv6FindAndFindFull 测试IPv6的FindFull方法
func TestIP2RegionIPv6FindAndFindFull(t *testing.T) {
	dbPath := getTestDBPath("ip2region_v6.xdb")

	ip2Region, err := NewIP2Region(iplocate.IPv6VersionNo, dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer ip2Region.Close()

	testIPs := []string{
		"2001:4860:4860::8888",
		"2001:4860:4860::8844",
		"2606:4700:4700::1111",
	}

	for _, ip := range testIPs {
		findFullResult := ip2Region.FindFull(ip)

		t.Logf("=== IP: %s ===", ip)
		t.Logf("FindFull 结果: Location=%s, Country=%s, Province=%s, City=%s, Area=%s, ISP=%s",
			findFullResult.Location, findFullResult.Country, findFullResult.Province, findFullResult.City, findFullResult.Area, findFullResult.ISP)
	}
}
