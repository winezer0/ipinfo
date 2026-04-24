package geolite2mmdb

import "testing"

// TestGeoLite2MMDBParseDBIPCityIPv4 测试解析DBIP IPv4城市数据库
func TestGeoLite2MMDBParseDBIPCityIPv4(t *testing.T) {
	dbPath := getTestDBPath("dbip-city-ipv4.mmdb")

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
		t.Logf("DBIP IPv4 - IP %s 查询结果: %s", ip, result.Location)
	}

	dbInfo := db.GetDatabaseInfo()
	if dbInfo == nil {
		t.Error("获取数据库信息失败")
	} else {
		t.Logf("数据库信息: %+v", dbInfo)
	}
}

// TestGeoLite2MMDBParseDBIPCityIPv6 测试解析DBIP IPv6城市数据库
func TestGeoLite2MMDBParseDBIPCityIPv6(t *testing.T) {
	dbPath := getTestDBPath("dbip-city-ipv6.mmdb")

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
		t.Logf("DBIP IPv6 - IP %s 查询结果: %s", ip, result.Location)
	}

	dbInfo := db.GetDatabaseInfo()
	if dbInfo == nil {
		t.Error("获取数据库信息失败")
	} else {
		t.Logf("数据库信息: %+v", dbInfo)
	}
}
