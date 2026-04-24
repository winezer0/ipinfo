// Package asncsv 提供对 CSV 格式 ASN 数据库的支持
package asncsv

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func getTestDBPath(dbname string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..", "..")
	return filepath.Join(projectRoot, "assets", dbname)
}

// TestASNCsvIPv4 测试CSV格式IPv4数据库查询
func TestASNCsvIPv4(t *testing.T) {
	dbPath := getTestDBPath("dbip-asn-ipv4.csv")

	querier := NewASNCsvQuerier(dbPath)
	if err := querier.Init(); err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer querier.Close()

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
	}

	for _, ip := range testIPs {
		result := querier.FindASN(ip)
		if result == nil {
			t.Errorf("查询IP %s 返回 nil", ip)
			continue
		}
		t.Logf("IP %s 查询结果: ASN=%d, 组织=%s, 找到ASN=%v",
			ip, result.OrganisationNumber, result.OrganisationName, result.FoundASN)
	}
}

// TestASNCsvIPv6 测试CSV格式IPv6数据库查询
func TestASNCsvIPv6(t *testing.T) {
	dbPath := getTestDBPath("dbip-asn-ipv6.csv")

	querier := NewASNCsvQuerier(dbPath)
	if err := querier.Init(); err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer querier.Close()

	testIPs := []string{
		"2001:4860:4860::8888",
		"2606:4700::6813",
	}

	for _, ip := range testIPs {
		result := querier.FindASN(ip)
		if result == nil {
			t.Errorf("查询IP %s 返回 nil", ip)
			continue
		}
		t.Logf("IP %s 查询结果: ASN=%d, 组织=%s, 找到ASN=%v",
			ip, result.OrganisationNumber, result.OrganisationName, result.FoundASN)
	}
}

// TestASNCsvBatchFind 测试批量查询
func TestASNCsvBatchFind(t *testing.T) {
	dbPath := getTestDBPath("dbip-asn-ipv4.csv")

	querier := NewASNCsvQuerier(dbPath)
	if err := querier.Init(); err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer querier.Close()

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
	}

	results := querier.BatchFindASN(testIPs)
	if len(results) != len(testIPs) {
		t.Errorf("期望结果数量为 %d，但得到了 %d", len(testIPs), len(results))
	}

	for _, ip := range testIPs {
		result := results[ip]
		if result == nil {
			t.Errorf("IP %s 的结果为 nil", ip)
			continue
		}
		t.Logf("批量查询IP %s 结果: ASN=%d, 组织=%s",
			result.IP, result.OrganisationNumber, result.OrganisationName)
	}
}

// TestASNCsvInvalidIP 测试无效IP查询
func TestASNCsvInvalidIP(t *testing.T) {
	dbPath := getTestDBPath("dbip-asn-ipv4.csv")

	querier := NewASNCsvQuerier(dbPath)
	if err := querier.Init(); err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer querier.Close()

	invalidIPs := []string{
		"invalid_ip",
		"",
		"999.999.999.999",
	}

	for _, ip := range invalidIPs {
		result := querier.FindASN(ip)
		if result == nil {
			t.Errorf("查询无效IP %s 返回 nil", ip)
			continue
		}
		if result.FoundASN {
			t.Errorf("查询无效IP %s 不应该找到ASN", ip)
		}
	}
}

// TestASNCsvEmptyPath 测试空路径初始化
func TestASNCsvEmptyPath(t *testing.T) {
	querier := NewASNCsvQuerier("")
	err := querier.Init()
	if err == nil {
		t.Error("期望初始化空路径失败，但成功了")
	}
}

// TestASNCsvNonExistentFile 测试不存在的文件
func TestASNCsvNonExistentFile(t *testing.T) {
	querier := NewASNCsvQuerier("/nonexistent/path/to/db.csv")
	err := querier.Init()
	if err == nil {
		t.Error("期望初始化不存在的文件失败，但成功了")
	}
}

// TestASNCsvASNToIPRanges 测试ASN反查IP范围
func TestASNCsvASNToIPRanges(t *testing.T) {
	dbPath := getTestDBPath("dbip-asn-ipv4.csv")

	querier := NewASNCsvQuerier(dbPath)
	if err := querier.Init(); err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer querier.Close()

	ipRanges, err := querier.ASNToIPRanges(13335)
	if err != nil {
		t.Errorf("ASN反查IP范围失败: %v", err)
		return
	}

	t.Logf("找到 %d 个IP范围", len(ipRanges))
	//for _, ipNet := range ipRanges {
	//	t.Logf("IP范围: %s", ipNet.String())
	//}
}

// TestASNCsvIsInitialized 测试初始化状态检查
func TestASNCsvIsInitialized(t *testing.T) {
	dbPath := getTestDBPath("dbip-asn-ipv4.csv")

	querier := NewASNCsvQuerier(dbPath)

	if querier.IsInitialized() {
		t.Error("未初始化时 IsInitialized 应该返回 false")
	}

	if err := querier.Init(); err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer querier.Close()

	if !querier.IsInitialized() {
		t.Error("初始化后 IsInitialized 应该返回 true")
	}
}

// TestASNCsvGetDatabaseInfo 测试获取数据库信息
func TestASNCsvGetDatabaseInfo(t *testing.T) {
	dbFiles := []struct {
		name     string
		filename string
	}{
		{name: "dbip-asn-ipv4", filename: "dbip-asn-ipv4.csv"},
		{name: "dbip-asn-ipv6", filename: "dbip-asn-ipv6.csv"},
	}

	for _, dbFile := range dbFiles {
		t.Run(dbFile.name, func(t *testing.T) {
			dbPath := getTestDBPath(dbFile.filename)

			querier := NewASNCsvQuerier(dbPath)

			// 测试初始化前的数据库信息
			infoBefore := querier.GetDatabaseInfo()
			if infoBefore == nil {
				t.Error("获取数据库信息失败，返回 nil")
				return
			}
			t.Logf("初始化前 - 类型: %s, 路径: %s, IPv4: %v, IPv6: %v",
				infoBefore.Type, infoBefore.DbPath, infoBefore.IsIPv4, infoBefore.IsIPv6)

			// 初始化数据库
			if err := querier.Init(); err != nil {
				t.Skipf("跳过测试，数据库文件加载失败: %v", err)
			}
			defer querier.Close()

			// 测试初始化后的数据库信息
			infoAfter := querier.GetDatabaseInfo()
			if infoAfter == nil {
				t.Error("获取数据库信息失败，返回 nil")
				return
			}

			t.Logf("初始化后 - 类型: %s, 路径: %s, IPv4: %v, IPv6: %v",
				infoAfter.Type, infoAfter.DbPath, infoAfter.IsIPv4, infoAfter.IsIPv6)

			// 验证数据库路径
			if infoAfter.DbPath != dbPath {
				t.Errorf("数据库路径不匹配，期望: %s, 实际: %s", dbPath, infoAfter.DbPath)
			}

			// 验证数据库类型
			if infoAfter.Type != "csv" {
				t.Errorf("数据库类型不正确，期望: csv, 实际: %s", infoAfter.Type)
			}
		})
	}
}

// TestASNCsv_ASNToIPRangesLazyLoad 测试ASNToIPRanges懒加载性能对比
func TestASNCsv_ASNToIPRangesLazyLoad(t *testing.T) {
	testCases := []struct {
		name   string
		dbFile string
		asn    uint64
	}{
		{name: "dbip-asn-ipv4-ASN13335", dbFile: "dbip-asn-ipv4.csv", asn: 13335},
		{name: "dbip-asn-ipv4-ASN15169", dbFile: "dbip-asn-ipv4.csv", asn: 15169},
		{name: "dbip-asn-ipv6-ASN13335", dbFile: "dbip-asn-ipv6.csv", asn: 13335},
		{name: "dbip-asn-ipv6-ASN15169", dbFile: "dbip-asn-ipv6.csv", asn: 15169},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbPath := getTestDBPath(tc.dbFile)

			querier := NewASNCsvQuerier(dbPath)
			if err := querier.Init(); err != nil {
				t.Skipf("跳过测试，数据库文件加载失败: %v", err)
			}
			defer querier.Close()

			// 首次调用（需要构建索引）
			start1 := time.Now()
			results1, err := querier.ASNToIPRanges(tc.asn)
			duration1 := time.Since(start1)
			if err != nil {
				t.Fatalf("首次ASNToIPRanges调用失败: %v", err)
			}

			// 二次调用（使用缓存索引）
			start2 := time.Now()
			results2, err := querier.ASNToIPRanges(tc.asn)
			duration2 := time.Since(start2)
			if err != nil {
				t.Fatalf("二次ASNToIPRanges调用失败: %v", err)
			}

			// 验证结果一致性
			if len(results1) != len(results2) {
				t.Errorf("两次查询结果数量不一致: 首次=%d, 二次=%d", len(results1), len(results2))
			}

			t.Logf("数据库: %s, 查询ASN: %d", tc.dbFile, tc.asn)
			t.Logf("找到 %d 个IP范围", len(results1))
			t.Logf("首次调用耗时: %v (构建索引)", duration1)
			t.Logf("二次调用耗时: %v (使用索引)", duration2)
			if duration1 > 0 && duration2 > 0 {
				speedup := float64(duration1) / float64(duration2)
				t.Logf("性能提升: %.2fx", speedup)
			}
		})
	}
}

// TestASNCsv_LazyLoadCombined 测试懒加载综合场景
func TestASNCsv_LazyLoadCombined(t *testing.T) {
	dbPath := getTestDBPath("dbip-asn-ipv4.csv")

	querier := NewASNCsvQuerier(dbPath)
	if err := querier.Init(); err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer querier.Close()

	t.Log("=== 阶段1: 首次调用ASNToIPRanges ===")
	start1 := time.Now()
	results, _ := querier.ASNToIPRanges(13335)
	t.Logf("ASNToIPRanges首次调用耗时: %v, 找到%d个IP范围", time.Since(start1), len(results))

	t.Log("=== 阶段2: 二次调用ASNToIPRanges ===")
	start2 := time.Now()
	results2, _ := querier.ASNToIPRanges(13335)
	t.Logf("ASNToIPRanges二次调用耗时: %v, 找到%d个IP范围", time.Since(start2), len(results2))

	t.Log("=== 阶段3: 查询不同ASN(15169) ===")
	start3 := time.Now()
	results3, _ := querier.ASNToIPRanges(15169)
	t.Logf("ASNToIPRanges查询ASN15169耗时: %v, 找到%d个IP范围", time.Since(start3), len(results3))

	if len(results) != len(results2) {
		t.Errorf("ASNToIPRanges结果数量不一致: %d vs %d", len(results), len(results2))
	}
}
