package queryip2

import (
	"github.com/winezer0/ipinfo/pkg/iplocate"
	"path/filepath"
	"runtime"
	"testing"
)

// getTestDBPath 获取测试数据库文件路径
func getTestDBPath(dbname string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")
	return filepath.Join(projectRoot, "assets", dbname)
}

// TestInitDBEngines_SingleDB 测试单数据库初始化
func TestInitDBEngines_SingleDB(t *testing.T) {
	config := &IPDbConfig{
		IpLocateDbs: []string{getTestDBPath("qqwry.dat")},
	}

	engine, err := InitDBEngines(config)
	if err != nil {
		t.Fatalf("初始化数据库引擎失败: %v", err)
	}
	defer engine.Close()

	if len(engine.Engines) != 1 {
		t.Errorf("期望 1 个引擎，得到 %d", len(engine.Engines))
	}

	t.Logf("引擎数量: %d", len(engine.Engines))
	t.Logf("引擎键: %v", getMapKeys(engine.Engines))
}

// TestInitDBEngines_MultipleDBs 测试多数据库初始化
func TestInitDBEngines_MultipleDBs(t *testing.T) {
	config := &IPDbConfig{
		IpLocateDbs: []string{
			getTestDBPath("qqwry.dat"),
			getTestDBPath("qqwry.ipdb"),
		},
	}

	engine, err := InitDBEngines(config)
	if err != nil {
		t.Fatalf("初始化数据库引擎失败: %v", err)
	}
	defer engine.Close()

	if len(engine.Engines) < 1 {
		t.Errorf("期望至少 1 个引擎，得到 %d", len(engine.Engines))
	}

	t.Logf("引擎数量: %d", len(engine.Engines))
	t.Logf("引擎键: %v", getMapKeys(engine.Engines))
}

// TestInitDBEngines_NoDB 测试无数据库配置
func TestInitDBEngines_NoDB(t *testing.T) {
	config := &IPDbConfig{
		IpLocateDbs: []string{},
	}

	_, err := InitDBEngines(config)
	if err == nil {
		t.Error("期望初始化失败，但成功了")
	}
}

// TestQuerySingleIP 测试单IP查询
func TestQuerySingleIP(t *testing.T) {
	config := &IPDbConfig{
		IpLocateDbs: []string{
			getTestDBPath("qqwry.dat"),
			getTestDBPath("qqwry.ipdb"),
		},
	}

	engine, err := InitDBEngines(config)
	if err != nil {
		t.Skipf("跳过测试，数据库加载失败: %v", err)
	}
	defer engine.Close()

	testIPs := []string{
		"8.8.8.8",
		"114.114.114.114",
		"1.1.1.1",
	}

	for _, ip := range testIPs {
		result := engine.QueryIP(ip)

		if result.IP != ip {
			t.Errorf("QueryIP(%s).IP = %s, want %s", ip, result.IP, ip)
		}

		t.Logf("IP: %s", ip)
		t.Logf("  数据库数量: %d", len(result.IPLocateResult))

		for source, locate := range result.IPLocateResult {
			t.Logf("  - 来源: %s", source)
			t.Logf("    位置: %s", locate.Location)
			t.Logf("    国家: %s, 省份: %s, 城市: %s", locate.Country, locate.Province, locate.City)
		}
	}
}

// TestQueryIPInfo 测试批量IP查询
func TestQueryIPInfo(t *testing.T) {
	config := &IPDbConfig{
		IpLocateDbs: []string{
			getTestDBPath("qqwry.dat"),
			getTestDBPath("qqwry.ipdb"),
		},
	}

	engine, err := InitDBEngines(config)
	if err != nil {
		t.Skipf("跳过测试，数据库加载失败: %v", err)
	}
	defer engine.Close()

	ipv4s := []string{
		"8.8.8.8",
		"114.114.114.114",
	}

	ipv6s := []string{
		"2001:db8::1",
	}

	info, err := engine.QueryIPInfo(ipv4s, ipv6s)
	if err != nil {
		t.Fatalf("查询IP信息失败: %v", err)
	}

	t.Logf("IPv4 结果数量: %d", len(info.IPv4Results))
	for _, result := range info.IPv4Results {
		t.Logf("  IP: %s, 数据库数量: %d", result.IP, len(result.IPLocateResult))
		for source, locate := range result.IPLocateResult {
			t.Logf("    - %s: %s", source, locate.Location)
		}
	}

	t.Logf("IPv6 结果数量: %d", len(info.IPv6Results))
}

// getMapKeys 获取map的所有键
func getMapKeys(m map[string]iplocate.IPInfo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
