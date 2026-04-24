package queryip

import (
	"fmt"
	"github.com/winezer0/ipinfo/pkg/asninfo"
	"github.com/winezer0/ipinfo/pkg/asninfo/asncsv"
	"github.com/winezer0/ipinfo/pkg/asninfo/asnmmdb"
	"github.com/winezer0/ipinfo/pkg/iplocate"
	"github.com/winezer0/ipinfo/pkg/iplocate/dbipmmdb"
	"github.com/winezer0/ipinfo/pkg/iplocate/geolite2mmdb"
	"github.com/winezer0/ipinfo/pkg/iplocate/ip2region"
	"github.com/winezer0/ipinfo/pkg/iplocate/ipdb"
	"github.com/winezer0/ipinfo/pkg/iplocate/qqwry"
	"github.com/winezer0/ipinfo/pkg/iplocate/zxwry"
	"path/filepath"
	"strings"

	"github.com/oschwald/maxminddb-golang"
)

// InitDBEngines 初始化所有数据库引擎
func InitDBEngines(config *IPDbConfig) (*DBEngine, error) {
	asnIPv4Manager, asnIPv6Manager, err := initASNManagers(config)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitASNFailed, err)
	}
	defer func() {
		if err != nil {
			closeASNManagers(asnIPv4Manager, asnIPv6Manager)
		}
	}()

	ipv4Engine, ipv6Engine, err := initIPEngines(config)
	if err != nil {
		closeASNManagers(asnIPv4Manager, asnIPv6Manager)
		return nil, fmt.Errorf("%w: %w", ErrInitIPFailed, err)
	}
	defer func() {
		if err != nil {
			if ipv4Engine != nil {
				ipv4Engine.Close()
			}
			if ipv6Engine != nil && ipv6Engine != ipv4Engine {
				ipv6Engine.Close()
			}
		}
	}()

	return &DBEngine{
		AsnIPv4Engine: asnIPv4Manager,
		AsnIPv6Engine: asnIPv6Manager,
		IPv4Engine:    ipv4Engine,
		IPv6Engine:    ipv6Engine,
	}, nil
}

// initASNManagers 初始化ASN数据库管理器
func initASNManagers(config *IPDbConfig) (asnQuerier4, asnQuerier6 asninfo.ASNQuerier, err error) {
	asnIPv4DbPath := config.AsnIpv4Db
	if asnIPv4DbPath == "" {
		asnIPv4DbPath = config.AsnIpvxDb
	}

	if asnIPv4DbPath != "" {
		asnQuerier4, err = createASNManager(asnIPv4DbPath)
		if err != nil {
			return nil, nil, fmt.Errorf("初始化IPv4 ASN数据库失败: %w", err)
		}
	}

	asnIPv6DbPath := config.AsnIpv6Db
	if asnIPv6DbPath == "" {
		asnIPv6DbPath = config.AsnIpvxDb
	}

	if asnIPv6DbPath != "" {
		if asnIPv6DbPath == asnIPv4DbPath && asnQuerier4 != nil {
			asnQuerier6 = asnQuerier4
		} else {
			asnQuerier6, err = createASNManager(asnIPv6DbPath)
			if err != nil {
				if asnQuerier4 != nil {
					asnQuerier4.Close()
				}
				return nil, nil, fmt.Errorf("初始化IPv6 ASN数据库失败: %w", err)
			}
		}
	}

	return asnQuerier4, asnQuerier6, nil
}

// createASNManager 创建并初始化单个ASN数据库管理器
func createASNManager(dbPath string) (asninfo.ASNQuerier, error) {
	ext := strings.ToLower(filepath.Ext(dbPath))

	switch ext {
	case ".mmdb":
		asnConfig := &asnmmdb.MMDBConfig{
			AsnIpvxDb:            dbPath,
			MaxConcurrentQueries: 100,
		}
		asnManager, err := asnmmdb.NewMMDBManager(asnConfig)
		if err != nil {
			return nil, err
		}
		if err := asnManager.Init(); err != nil {
			return nil, err
		}
		return asnManager, nil

	case ".csv":
		asnManager := asncsv.NewASNCsvQuerier(dbPath)
		if err := asnManager.Init(); err != nil {
			return nil, err
		}
		return asnManager, nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedASNFormat, ext)
	}
}

// closeASNManagers 关闭所有ASN数据库管理器
func closeASNManagers(asnIPv4Manager, asnIPv6Manager asninfo.ASNQuerier) {
	if asnIPv4Manager != nil {
		asnIPv4Manager.Close()
	}
	if asnIPv6Manager != nil && asnIPv6Manager != asnIPv4Manager {
		asnIPv6Manager.Close()
	}
}

// initIPEngines 初始化IP地理位置数据库
func initIPEngines(config *IPDbConfig) (iplocate.IPInfo, iplocate.IPInfo, error) {
	ipv4DbPath := config.Ipv4LocateDb
	if ipv4DbPath == "" {
		ipv4DbPath = config.IpvxLocateDb
	}

	if ipv4DbPath == "" {
		return nil, nil, ErrMissingIPv4Config
	}

	ipv4Engine, err := createIPEngine(ipv4DbPath, iplocate.IPv4VersionNo)
	if err != nil {
		return nil, nil, fmt.Errorf("初始化IPv4数据库失败: %w", err)
	}

	ipv6DbPath := config.Ipv6LocateDb
	if ipv6DbPath == "" {
		ipv6DbPath = config.IpvxLocateDb
	}

	if ipv6DbPath == "" {
		return nil, nil, ErrMissingIPv6Config
	}

	var ipv6Engine iplocate.IPInfo
	if ipv6DbPath == ipv4DbPath {
		ipv6Engine = ipv4Engine
	} else {
		ipv6Engine, err = createIPEngine(ipv6DbPath, iplocate.IPv6VersionNo)
		if err != nil {
			ipv4Engine.Close()
			return nil, nil, fmt.Errorf("初始化IPv6数据库失败: %w", err)
		}
	}

	return ipv4Engine, ipv6Engine, nil
}

// createIPEngine 根据文件后缀创建对应的IP数据库引擎
func createIPEngine(dbPath string, version int) (iplocate.IPInfo, error) {
	ext := strings.ToLower(filepath.Ext(dbPath))

	switch ext {
	case ".xdb":
		return createIP2RegionEngine(dbPath)

	case ".mmdb":
		return createMMDBEngine(dbPath, version)

	case ".ipdb":
		return createIPDBEngine(dbPath)

	case ".db", ".dat":
		return createWryEngine(dbPath, version)

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
func createMMDBEngine(dbPath string, version int) (iplocate.IPInfo, error) {
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
func createWryEngine(dbPath string, version int) (iplocate.IPInfo, error) {
	if version == iplocate.IPv4VersionNo {
		engine, err := qqwry.NewQQWryDB(dbPath)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInitQQWryFailed, err)
		}
		return engine, nil
	}

	engine, err := zxwry.NewZXWryDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitZXWryFailed, err)
	}
	return engine, nil
}
