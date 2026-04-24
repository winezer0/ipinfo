package queryip2

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/winezer0/ipinfo/pkg/iplocate"
	"github.com/winezer0/ipinfo/pkg/iplocate/dbipmmdb"
	"github.com/winezer0/ipinfo/pkg/iplocate/geolite2mmdb"
	"github.com/winezer0/ipinfo/pkg/iplocate/ip2region"
	"github.com/winezer0/ipinfo/pkg/iplocate/ipdb"
	"github.com/winezer0/ipinfo/pkg/iplocate/qqwry"
	"github.com/winezer0/ipinfo/pkg/iplocate/zxwry"

	"github.com/oschwald/maxminddb-golang"
)

// InitDBEngines 初始化所有数据库引擎（支持多数据库）
func InitDBEngines(config *IPDbConfig) (*DBEngine, error) {
	engines, err := initIPEngines(config)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitIPFailed, err)
	}

	return &DBEngine{
		Engines: engines,
	}, nil
}

// initIPEngines 初始化多个IP地理位置数据库
func initIPEngines(config *IPDbConfig) (engines map[string]iplocate.IPInfo, err error) {
	if len(config.IpLocateDbs) == 0 {
		return nil, ErrMissingIPConfig
	}

	engines = make(map[string]iplocate.IPInfo)

	for _, dbPath := range config.IpLocateDbs {
		engine, err := createIPEngine(dbPath)
		if err != nil {
			closeIPEngines(engines)
			return nil, fmt.Errorf("初始化数据库 %s 失败: %w", dbPath, err)
		}

		source := Source(dbPath)
		engines[string(source)] = engine
	}

	if len(engines) == 0 {
		closeIPEngines(engines)
		return nil, ErrMissingIPConfig
	}

	return engines, nil
}

// createIPEngine 根据文件后缀创建对应的IP数据库引擎
func createIPEngine(dbPath string) (iplocate.IPInfo, error) {
	ext := strings.ToLower(filepath.Ext(dbPath))

	switch ext {
	case ".xdb":
		return createIP2RegionEngine(dbPath)

	case ".mmdb":
		return createMMDBEngine(dbPath)

	case ".ipdb":
		return createIPDBEngine(dbPath)

	case ".db", ".dat":
		return createWryEngine(dbPath)

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedIPFormat, ext)
	}
}

// createIP2RegionEngine 创建IP2Region引擎
func createIP2RegionEngine(dbPath string) (iplocate.IPInfo, error) {
	engine, err := ip2region.NewIP2Region(iplocate.IPv4VersionNo, dbPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitIP2RegionFailed, err)
	}
	return engine, nil
}

// createMMDBEngine 创建MMDB引擎
func createMMDBEngine(dbPath string) (iplocate.IPInfo, error) {
	db, err := maxminddb.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOpenMMDBFailed, err)
	}

	dbType := db.Metadata.DatabaseType
	db.Close()

	if strings.Contains(strings.ToLower(dbType), "dbip") {
		return createDBIPEngine(dbPath)
	}
	return createGeoLite2Engine(dbPath)
}

// createDBIPEngine 创建DBIP引擎
func createDBIPEngine(dbPath string) (iplocate.IPInfo, error) {
	engine, err := dbipmmdb.NewDBIPMMDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitDBIPFailed, err)
	}
	return engine, nil
}

// createGeoLite2Engine 创建GeoLite2引擎
func createGeoLite2Engine(dbPath string) (iplocate.IPInfo, error) {
	engine, err := geolite2mmdb.NewGeoLite2MMDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitGeoLite2Failed, err)
	}
	return engine, nil
}

// createIPDBEngine 创建IPDB引擎
func createIPDBEngine(dbPath string) (iplocate.IPInfo, error) {
	engine, err := ipdb.NewIPDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitIPDBFailed, err)
	}
	return engine, nil
}

// createWryEngine 创建纯真数据库引擎
func createWryEngine(dbPath string) (iplocate.IPInfo, error) {
	engine, err := qqwry.NewQQWryDB(dbPath)
	if err != nil {
		engine6, err6 := zxwry.NewZXWryDB(dbPath)
		if err6 != nil {
			return nil, fmt.Errorf("%w: %w", ErrInitQQWryFailed, err)
		}
		return engine6, nil
	}
	return engine, nil
}

// closeIPEngines 关闭所有IP数据库引擎
func closeIPEngines(engines map[string]iplocate.IPInfo) {
	for _, engine := range engines {
		if engine != nil {
			engine.Close()
		}
	}
}
