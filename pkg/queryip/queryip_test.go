package queryip

import (
	"github.com/winezer0/ipinfo/pkg/utils"
	"path/filepath"
	"runtime"
	"testing"
)

func getTestDBPath(dbname string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")
	return filepath.Join(projectRoot, "assets", dbname)
}

func TestQueryIP(t *testing.T) {
	asnIpvxDb := getTestDBPath("geolite2-asn.mmdb")
	ipv4LocateDb := getTestDBPath("qqwry.dat")
	ipv6LocateDb := getTestDBPath("zxipv6wry.db")

	config := &IPDbConfig{
		AsnIpvxDb:    asnIpvxDb,
		Ipv4LocateDb: ipv4LocateDb,
		Ipv6LocateDb: ipv6LocateDb,
	}

	engine, err := InitDBEngines(config)
	if err != nil {
		t.Skipf("跳过测试：无法初始化数据库引擎: %v", err)
		return
	}
	defer engine.Close()

	t.Run("TestQuerySingleIP", func(t *testing.T) {
		result4 := engine.QueryIP("1.1.1.1")
		t.Logf("IPv4查询结果 - IP: %s, 位置: %+v, ASN: %+v", result4.IP, result4.IPLocate, result4.ASNInfo)

		result6 := engine.QueryIP("2001:4860:4860::8888")
		t.Logf("IPv6查询结果 - IP: %s, 位置: %+v, ASN: %+v", result6.IP, result6.IPLocate, result6.ASNInfo)
	})

	t.Run("TestQueryIPInfo", func(t *testing.T) {
		ipv4s := []string{"8.8.8.8", "1.1.1.1"}
		ipv6s := []string{"2001:4860:4860::8888", "2606:4700:4700::1111"}

		ipInfo, err := engine.QueryIPInfo(ipv4s, ipv6s)
		if err != nil {
			t.Errorf("批量查询失败: %v", err)
			return
		}

		t.Logf("IPv4位置信息: %+v", utils.ToJSON(ipInfo.IPv4Locations))
		t.Logf("IPv6位置信息: %+v", utils.ToJSON(ipInfo.IPv6Locations))
		t.Logf("IPv4 ASN信息: %+v", utils.ToJSON(ipInfo.IPv4AsnInfos))
		t.Logf("IPv6 ASN信息: %+v", utils.ToJSON(ipInfo.IPv6AsnInfos))

		if len(ipInfo.IPv4Locations) != len(ipv4s) {
			t.Errorf("IPv4位置信息数量不匹配，期望: %d, 实际: %d", len(ipv4s), len(ipInfo.IPv4Locations))
		}

		if len(ipInfo.IPv6Locations) != len(ipv6s) {
			t.Errorf("IPv6位置信息数量不匹配，期望: %d, 实际: %d", len(ipv6s), len(ipInfo.IPv6Locations))
		}

		if len(ipInfo.IPv4AsnInfos) != len(ipv4s) {
			t.Errorf("IPv4 ASN信息数量不匹配，期望: %d, 实际: %d", len(ipv4s), len(ipInfo.IPv4AsnInfos))
		}

		if len(ipInfo.IPv6AsnInfos) != len(ipv6s) {
			t.Errorf("IPv6 ASN信息数量不匹配，期望: %d, 实际: %d", len(ipv6s), len(ipInfo.IPv6AsnInfos))
		}
	})
}

func TestCreateASNManagerMMDB(t *testing.T) {
	dbFiles := []string{
		"dbip-asn-ipv4.mmdb",
		"dbip-asn-ipv6.mmdb",
		"dbip-asn.mmdb",
		"geolite2-asn-ipv4.mmdb",
		"geolite2-asn-ipv6.mmdb",
		"geolite2-asn.mmdb",
	}

	for _, dbFile := range dbFiles {
		t.Run(dbFile, func(t *testing.T) {
			dbPath := getTestDBPath(dbFile)

			manager, err := createASNManager(dbPath)
			if err != nil {
				t.Fatalf("创建ASN管理器失败: %v", err)
			}
			defer manager.Close()

			if !manager.IsInitialized() {
				t.Error("ASN管理器未正确初始化")
			}

			dbInfo := manager.GetDatabaseInfo()
			if dbInfo == nil {
				t.Error("获取数据库信息失败")
				return
			}

			t.Logf("数据库信息 - 类型: %s, 路径: %s, IPv4: %v, IPv6: %v",
				dbInfo.Type, dbInfo.DbPath, dbInfo.IsIPv4, dbInfo.IsIPv6)

			testIP := "8.8.8.8"
			if dbInfo.IsIPv6 && !dbInfo.IsIPv4 {
				testIP = "2001:4860:4860::8888"
			}

			result := manager.FindASN(testIP)
			if result == nil {
				t.Errorf("查询IP %s 返回 nil", testIP)
			} else {
				t.Logf("IP %s 查询结果 - ASN: %d, 组织: %s", testIP, result.OrganisationNumber, result.OrganisationName)
			}
		})
	}
}

func TestCreateASNManagerCSV(t *testing.T) {
	dbFiles := []string{
		"dbip-asn-ipv4.csv",
		"dbip-asn-ipv6.csv",
	}

	for _, dbFile := range dbFiles {
		t.Run(dbFile, func(t *testing.T) {
			dbPath := getTestDBPath(dbFile)

			manager, err := createASNManager(dbPath)
			if err != nil {
				t.Fatalf("创建ASN管理器失败: %v", err)
			}
			defer manager.Close()

			if !manager.IsInitialized() {
				t.Error("ASN管理器未正确初始化")
			}

			dbInfo := manager.GetDatabaseInfo()
			if dbInfo == nil {
				t.Error("获取数据库信息失败")
				return
			}

			t.Logf("数据库信息 - 类型: %s, 路径: %s, IPv4: %v, IPv6: %v",
				dbInfo.Type, dbInfo.DbPath, dbInfo.IsIPv4, dbInfo.IsIPv6)

			if dbInfo.Type != "csv" {
				t.Errorf("数据库类型不正确，期望: csv, 实际: %s", dbInfo.Type)
			}

			testIP := "8.8.8.8"
			if dbInfo.IsIPv6 && !dbInfo.IsIPv4 {
				testIP = "2001:4860:4860::8888"
			}

			result := manager.FindASN(testIP)
			if result == nil {
				t.Errorf("查询IP %s 返回 nil", testIP)
			} else {
				t.Logf("IP %s 查询结果 - ASN: %d, 组织: %s", testIP, result.OrganisationNumber, result.OrganisationName)
			}
		})
	}
}

func TestCreateASNManagerUnsupportedFormat(t *testing.T) {
	invalidFiles := []string{
		"test.txt",
		"test.json",
		"test.xml",
	}

	for _, file := range invalidFiles {
		t.Run(file, func(t *testing.T) {
			_, err := createASNManager(file)
			if err == nil {
				t.Error("期望创建失败，但成功了")
			}
			t.Logf("预期错误: %v", err)
		})
	}
}

func TestInitDBEnginesWithCSVASN(t *testing.T) {
	config := &IPDbConfig{
		AsnIpv4Db:    getTestDBPath("dbip-asn-ipv4.csv"),
		AsnIpv6Db:    getTestDBPath("dbip-asn-ipv6.csv"),
		Ipv4LocateDb: getTestDBPath("qqwry.dat"),
		Ipv6LocateDb: getTestDBPath("zxipv6wry.db"),
	}

	engine, err := InitDBEngines(config)
	if err != nil {
		t.Fatalf("初始化数据库引擎失败: %v", err)
	}
	defer engine.Close()

	if engine.AsnIPv4Engine == nil {
		t.Error("IPv4 ASN引擎未初始化")
	} else {
		t.Logf("IPv4 ASN引擎已初始化")
		dbInfo := engine.AsnIPv4Engine.GetDatabaseInfo()
		if dbInfo != nil {
			t.Logf("IPv4 ASN数据库信息 - 类型: %s, 路径: %s", dbInfo.Type, dbInfo.DbPath)
		}
	}

	if engine.AsnIPv6Engine == nil {
		t.Error("IPv6 ASN引擎未初始化")
	} else {
		t.Logf("IPv6 ASN引擎已初始化")
		dbInfo := engine.AsnIPv6Engine.GetDatabaseInfo()
		if dbInfo != nil {
			t.Logf("IPv6 ASN数据库信息 - 类型: %s, 路径: %s", dbInfo.Type, dbInfo.DbPath)
		}
	}

	result4 := engine.QueryIP("8.8.8.8")
	t.Logf("IPv4查询结果 - IP: %s, 位置: %+v, ASN: %+v", result4.IP, result4.IPLocate, result4.ASNInfo)

	result6 := engine.QueryIP("2001:4860:4860::8888")
	t.Logf("IPv6查询结果 - IP: %s, 位置: %+v, ASN: %+v", result6.IP, result6.IPLocate, result6.ASNInfo)
}

func TestInitDBEnginesWithMMDBASN(t *testing.T) {
	config := &IPDbConfig{
		AsnIpvxDb:    getTestDBPath("geolite2-asn.mmdb"),
		Ipv4LocateDb: getTestDBPath("qqwry.dat"),
		Ipv6LocateDb: getTestDBPath("zxipv6wry.db"),
	}

	engine, err := InitDBEngines(config)
	if err != nil {
		t.Fatalf("初始化数据库引擎失败: %v", err)
	}
	defer engine.Close()

	if engine.AsnIPv4Engine == nil {
		t.Error("IPv4 ASN引擎未初始化")
	} else {
		t.Logf("IPv4 ASN引擎已初始化")
		dbInfo := engine.AsnIPv4Engine.GetDatabaseInfo()
		if dbInfo != nil {
			t.Logf("IPv4 ASN数据库信息 - 类型: %s, 路径: %s", dbInfo.Type, dbInfo.DbPath)
		}
	}

	result := engine.QueryIP("8.8.8.8")
	t.Logf("IPv4查询结果 - IP: %s, 位置: %+v, ASN: %+v", result.IP, result.IPLocate, result.ASNInfo)
}

func TestInitDBEnginesWithMixedASN(t *testing.T) {
	config := &IPDbConfig{
		AsnIpv4Db:    getTestDBPath("dbip-asn-ipv4.csv"),
		AsnIpv6Db:    getTestDBPath("geolite2-asn-ipv6.mmdb"),
		Ipv4LocateDb: getTestDBPath("qqwry.dat"),
		Ipv6LocateDb: getTestDBPath("zxipv6wry.db"),
	}

	engine, err := InitDBEngines(config)
	if err != nil {
		t.Fatalf("初始化数据库引擎失败: %v", err)
	}
	defer engine.Close()

	if engine.AsnIPv4Engine == nil {
		t.Error("IPv4 ASN引擎未初始化")
	} else {
		t.Logf("IPv4 ASN引擎已初始化")
		dbInfo := engine.AsnIPv4Engine.GetDatabaseInfo()
		if dbInfo != nil {
			t.Logf("IPv4 ASN数据库信息 - 类型: %s, 路径: %s", dbInfo.Type, dbInfo.DbPath)
		}
	}

	if engine.AsnIPv6Engine == nil {
		t.Error("IPv6 ASN引擎未初始化")
	} else {
		t.Logf("IPv6 ASN引擎已初始化")
		dbInfo := engine.AsnIPv6Engine.GetDatabaseInfo()
		if dbInfo != nil {
			t.Logf("IPv6 ASN数据库信息 - 类型: %s, 路径: %s", dbInfo.Type, dbInfo.DbPath)
		}
	}

	result4 := engine.QueryIP("8.8.8.8")
	t.Logf("IPv4查询结果 - IP: %s, 位置: %+v, ASN: %+v", result4.IP, result4.IPLocate, result4.ASNInfo)

	result6 := engine.QueryIP("2001:4860:4860::8888")
	t.Logf("IPv6查询结果 - IP: %s, 位置: %+v, ASN: %+v", result6.IP, result6.IPLocate, result6.ASNInfo)
}
