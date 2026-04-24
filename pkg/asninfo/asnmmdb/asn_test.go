package asnmmdb

import (
	"net"
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

func TestMMDBManager_FindASN(t *testing.T) {
	asnIpvxDb := getTestDBPath("geolite2-asn.mmdb")
	config := &MMDBConfig{
		AsnIpvxDb:            asnIpvxDb,
		MaxConcurrentQueries: 100,
	}

	manager, err := NewMMDBManager(config)
	if err != nil {
		t.Fatalf("创建数据库管理器失败: %v", err)
	}

	if err := manager.InitMMDBConn(); err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
	}
	defer manager.Close()

	// 定义测试IP列表
	testIPs := []string{
		"8.8.8.8",         // Google DNS (IPv4)
		"2606:4700::6813", // Cloudflare (IPv6)
		"192.168.1.1",     // 内网地址
		"1.1.1.1",         // Cloudflare
		"116.162.1.1",     // Cloudflare
	}

	// 测试单个IP查询
	t.Run("单个IP查询测试", func(t *testing.T) {
		for _, ipStr := range testIPs {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				t.Errorf("无效的IP地址: %s", ipStr)
				continue
			}

			ipInfo := manager.FindASN(ipStr)
			if ipInfo == nil {
				t.Errorf("无法解析IP信息: %s", ipStr)
				continue
			}

			// 直接打印结果，而不是依赖logging包
			t.Logf("IP: %s | 版本: %d | 找到ASN: %v | ASN: %d | 组织: %s",
				ipInfo.IP,
				ipInfo.IPVersion,
				ipInfo.FoundASN,
				ipInfo.OrganisationNumber,
				ipInfo.OrganisationName,
			)
		}
	})
}

func TestMMDBManager_ASNToIPRanges(t *testing.T) {
	asnIpvxDb := getTestDBPath("geolite2-asn.mmdb")
	config := &MMDBConfig{
		AsnIpvxDb:            asnIpvxDb,
		MaxConcurrentQueries: 100,
	}

	manager, err := NewMMDBManager(config)
	if err != nil {
		t.Fatalf("创建数据库管理器失败: %v", err)
	}

	if err := manager.InitMMDBConn(); err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
	}
	defer manager.Close()

	// 测试ASN到IP范围查询
	t.Run("ASN到IP范围查询测试", func(t *testing.T) {
		results, err := manager.ASNToIPRanges(13335)
		if err != nil {
			t.Errorf("ASN到IP范围查询失败: %v", err)
			return
		}
		t.Logf("找到 %d 个IP范围", len(results))
		//for _, ipNet := range results {
		//	t.Logf("IP范围: %s", ipNet.String())
		//}
	})
}

func TestMMDBManager_BatchFindASN(t *testing.T) {
	asnIpvxDb := getTestDBPath("geolite2-asn.mmdb")
	config := &MMDBConfig{
		AsnIpvxDb:            asnIpvxDb,
		MaxConcurrentQueries: 100,
	}

	manager, err := NewMMDBManager(config)
	if err != nil {
		t.Fatalf("创建数据库管理器失败: %v", err)
	}

	// 初始化数据库连接
	if err := manager.InitMMDBConn(); err != nil {
		t.Skipf("跳过测试，因为无法加载数据库: %v", err)
	}
	defer manager.Close()

	// 验证数据库是否已初始化
	if !manager.IsInitialized() {
		t.Fatal("数据库未正确初始化")
	}

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
		"2001:db8::1",
		"invalid_ip",
		"",
	}

	results := manager.BatchFindASN(testIPs)

	if len(results) != len(testIPs) {
		t.Errorf("期望结果数量为 %d，但得到了 %d", len(testIPs), len(results))
	}

	for _, ip := range testIPs {
		result := results[ip]
		if result == nil {
			t.Errorf("IP %s 的结果为 nil", ip)
			continue
		}

		// 直接打印结果，而不是依赖logging包
		t.Logf("IP: %s, 版本: %d, 找到ASN: %v, ASN: %d, 组织: %s",
			result.IP, result.IPVersion, result.FoundASN, result.OrganisationNumber, result.OrganisationName)
	}
}

// TestAllASNDatabases 测试所有ASN数据库文件的支持情况
func TestAllASNDatabases(t *testing.T) {
	// 定义所有ASN数据库文件
	dbFiles := []struct {
		name     string
		filename string
	}{
		{name: "dbip-asn-ipv4", filename: "dbip-asn-ipv4.mmdb"},
		{name: "dbip-asn-ipv6", filename: "dbip-asn-ipv6.mmdb"},
		{name: "dbip-asn", filename: "dbip-asn.mmdb"},
		{name: "geolite2-asn-ipv4", filename: "geolite2-asn-ipv4.mmdb"},
		{name: "geolite2-asn-ipv6", filename: "geolite2-asn-ipv6.mmdb"},
		{name: "geolite2-asn", filename: "geolite2-asn.mmdb"},
	}

	// 定义测试IP列表
	testIPs := []struct {
		ip      string
		version int
	}{
		{ip: "8.8.8.8", version: 4},
		{ip: "1.1.1.1", version: 4},
		{ip: "114.114.114.114", version: 4},
		{ip: "2001:4860:4860::8888", version: 6},
		{ip: "2606:4700::6813", version: 6},
	}

	for _, dbFile := range dbFiles {
		t.Run(dbFile.name, func(t *testing.T) {
			dbPath := getTestDBPath(dbFile.filename)

			// 创建配置
			config := &MMDBConfig{
				AsnIpvxDb:            dbPath,
				MaxConcurrentQueries: 100,
			}

			manager, err := NewMMDBManager(config)
			if err != nil {
				t.Skipf("跳过测试，创建管理器失败: %v", err)
			}

			if err := manager.InitMMDBConn(); err != nil {
				t.Skipf("跳过测试，数据库文件加载失败: %v", err)
			}
			defer manager.Close()

			// 测试每个IP
			for _, testIP := range testIPs {
				t.Run(testIP.ip, func(t *testing.T) {
					result := manager.FindASN(testIP.ip)
					if result == nil {
						t.Logf("IP %s 查询结果为 nil", testIP.ip)
						return
					}

					t.Logf("IP: %s | 版本: %d | 找到ASN: %v | ASN: %d | 组织: %s",
						result.IP,
						result.IPVersion,
						result.FoundASN,
						result.OrganisationNumber,
						result.OrganisationName,
					)
				})
			}
		})
	}
}

// TestASNDatabasesIPv4Only 测试仅支持IPv4的数据库
func TestASNDatabasesIPv4Only(t *testing.T) {
	ipv4DBs := []string{
		"dbip-asn-ipv4.mmdb",
		"geolite2-asn-ipv4.mmdb",
	}

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
	}

	for _, dbFile := range ipv4DBs {
		t.Run(dbFile, func(t *testing.T) {
			dbPath := getTestDBPath(dbFile)

			config := &MMDBConfig{
				AsnIpvxDb:            dbPath,
				MaxConcurrentQueries: 100,
			}

			manager, err := NewMMDBManager(config)
			if err != nil {
				t.Skipf("跳过测试，创建管理器失败: %v", err)
			}
			if err := manager.InitMMDBConn(); err != nil {
				t.Skipf("跳过测试，数据库文件加载失败: %v", err)
			}
			defer manager.Close()

			t.Logf("数据库: %s", dbFile)

			for _, ip := range testIPs {
				result := manager.FindASN(ip)
				if result != nil {
					t.Logf("  %s -> ASN: %d, 组织: %s", ip, result.OrganisationNumber, result.OrganisationName)
				}
			}
		})
	}
}

// TestASNDatabasesIPv6Only 测试仅支持IPv6的数据库
func TestASNDatabasesIPv6Only(t *testing.T) {
	ipv6DBs := []string{
		"dbip-asn-ipv6.mmdb",
		"geolite2-asn-ipv6.mmdb",
	}

	testIPs := []string{
		"2001:4860:4860::8888",
		"2606:4700::6813",
	}

	for _, dbFile := range ipv6DBs {
		t.Run(dbFile, func(t *testing.T) {
			dbPath := getTestDBPath(dbFile)

			config := &MMDBConfig{
				AsnIpvxDb:            dbPath,
				MaxConcurrentQueries: 100,
			}

			manager, err := NewMMDBManager(config)
			if err != nil {
				t.Skipf("跳过测试，创建管理器失败: %v", err)
			}
			if err := manager.InitMMDBConn(); err != nil {
				t.Skipf("跳过测试，数据库文件加载失败: %v", err)
			}
			defer manager.Close()

			t.Logf("数据库: %s", dbFile)

			for _, ip := range testIPs {
				result := manager.FindASN(ip)
				if result != nil {
					t.Logf("  %s -> ASN: %d, 组织: %s", ip, result.OrganisationNumber, result.OrganisationName)
				}
			}
		})
	}
}

// TestASNDatabasesDualStack 测试支持双栈的数据库
func TestASNDatabasesDualStack(t *testing.T) {
	dualDBs := []string{
		"dbip-asn.mmdb",
		"geolite2-asn.mmdb",
	}

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"2001:4860:4860::8888",
		"2606:4700::6813",
	}

	for _, dbFile := range dualDBs {
		t.Run(dbFile, func(t *testing.T) {
			dbPath := getTestDBPath(dbFile)

			config := &MMDBConfig{
				AsnIpvxDb:            dbPath,
				MaxConcurrentQueries: 100,
			}

			manager, err := NewMMDBManager(config)
			if err != nil {
				t.Skipf("跳过测试，创建管理器失败: %v", err)
			}
			if err := manager.InitMMDBConn(); err != nil {
				t.Skipf("跳过测试，数据库文件加载失败: %v", err)
			}
			defer manager.Close()

			t.Logf("数据库: %s", dbFile)

			for _, ip := range testIPs {
				result := manager.FindASN(ip)
				if result != nil {
					t.Logf("  %s -> ASN: %d, 组织: %s", ip, result.OrganisationNumber, result.OrganisationName)
				}
			}
		})
	}
}

// TestMMDBManager_GetDatabaseInfo 测试获取数据库信息
func TestMMDBManager_GetDatabaseInfo(t *testing.T) {
	dbFiles := []struct {
		name     string
		filename string
	}{
		{name: "dbip-asn-ipv4", filename: "dbip-asn-ipv4.mmdb"},
		{name: "dbip-asn-ipv6", filename: "dbip-asn-ipv6.mmdb"},
		{name: "dbip-asn", filename: "dbip-asn.mmdb"},
		{name: "geolite2-asn-ipv4", filename: "geolite2-asn-ipv4.mmdb"},
		{name: "geolite2-asn-ipv6", filename: "geolite2-asn-ipv6.mmdb"},
		{name: "geolite2-asn", filename: "geolite2-asn.mmdb"},
	}

	for _, dbFile := range dbFiles {
		t.Run(dbFile.name, func(t *testing.T) {
			dbPath := getTestDBPath(dbFile.filename)

			config := &MMDBConfig{
				AsnIpvxDb:            dbPath,
				MaxConcurrentQueries: 100,
			}

			manager, err := NewMMDBManager(config)
			if err != nil {
				t.Skipf("跳过测试，创建管理器失败: %v", err)
			}

			// 测试初始化前的数据库信息
			infoBefore := manager.GetDatabaseInfo()
			if infoBefore == nil {
				t.Error("获取数据库信息失败，返回 nil")
				return
			}
			t.Logf("初始化前 - 类型: %s, 路径: %s, IPv4: %v, IPv6: %v",
				infoBefore.Type, infoBefore.DbPath, infoBefore.IsIPv4, infoBefore.IsIPv6)

			// 初始化数据库
			if err := manager.InitMMDBConn(); err != nil {
				t.Skipf("跳过测试，数据库文件加载失败: %v", err)
			}
			defer manager.Close()

			// 测试初始化后的数据库信息
			infoAfter := manager.GetDatabaseInfo()
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
		})
	}
}

// TestMMDBManager_ASNToIPRangesLazyLoad 测试ASNToIPRanges懒加载性能对比
func TestMMDBManager_ASNToIPRangesLazyLoad(t *testing.T) {
	testCases := []struct {
		name   string
		dbFile string
		asn    uint64
	}{
		{name: "geolite2-asn-ASN13335", dbFile: "geolite2-asn.mmdb", asn: 13335},
		{name: "geolite2-asn-ASN15169", dbFile: "geolite2-asn.mmdb", asn: 15169},
		{name: "dbip-asn-ASN13335", dbFile: "dbip-asn.mmdb", asn: 13335},
		{name: "dbip-asn-ASN15169", dbFile: "dbip-asn.mmdb", asn: 15169},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbPath := getTestDBPath(tc.dbFile)

			config := &MMDBConfig{
				AsnIpvxDb:            dbPath,
				MaxConcurrentQueries: 100,
			}

			manager, err := NewMMDBManager(config)
			if err != nil {
				t.Skipf("跳过测试，创建管理器失败: %v", err)
			}

			if err := manager.InitMMDBConn(); err != nil {
				t.Skipf("跳过测试，数据库文件加载失败: %v", err)
			}
			defer manager.Close()

			// 首次调用（需要遍历数据库+构建索引）
			start1 := time.Now()
			results1, err := manager.ASNToIPRanges(tc.asn)
			duration1 := time.Since(start1)
			if err != nil {
				t.Fatalf("首次ASNToIPRanges调用失败: %v", err)
			}

			// 二次调用（使用缓存索引）
			start2 := time.Now()
			results2, err := manager.ASNToIPRanges(tc.asn)
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
			t.Logf("首次调用耗时: %v (遍历数据库+构建索引)", duration1)
			t.Logf("二次调用耗时: %v (使用索引)", duration2)
			if duration1 > 0 {
				speedup := float64(duration1) / float64(duration2)
				t.Logf("性能提升: %.2fx", speedup)
			}
		})
	}
}

// TestMMDBManager_LazyLoadCombined 测试懒加载综合场景
func TestMMDBManager_LazyLoadCombined(t *testing.T) {
	dbPath := getTestDBPath("geolite2-asn.mmdb")

	config := &MMDBConfig{
		AsnIpvxDb:            dbPath,
		MaxConcurrentQueries: 100,
	}

	manager, err := NewMMDBManager(config)
	if err != nil {
		t.Skipf("跳过测试，创建管理器失败: %v", err)
	}

	if err := manager.InitMMDBConn(); err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer manager.Close()

	t.Log("=== 阶段1: 首次调用ASNToIPRanges ===")
	start1 := time.Now()
	results, _ := manager.ASNToIPRanges(13335)
	t.Logf("ASNToIPRanges首次调用耗时: %v, 找到%d个IP范围", time.Since(start1), len(results))

	t.Log("=== 阶段2: 二次调用ASNToIPRanges ===")
	start2 := time.Now()
	results2, _ := manager.ASNToIPRanges(13335)
	t.Logf("ASNToIPRanges二次调用耗时: %v, 找到%d个IP范围", time.Since(start2), len(results2))

	t.Log("=== 阶段3: 查询不同ASN(15169) ===")
	start3 := time.Now()
	results3, _ := manager.ASNToIPRanges(15169)
	t.Logf("ASNToIPRanges查询ASN15169耗时: %v, 找到%d个IP范围", time.Since(start3), len(results3))

	if len(results) != len(results2) {
		t.Errorf("ASNToIPRanges结果数量不一致: %d vs %d", len(results), len(results2))
	}
}
