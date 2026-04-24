package queryip2

import (
	"github.com/winezer0/ipinfo/pkg/iplocate"
)

// IPDbConfig 存储程序配置（支持多数据库）
type IPDbConfig struct {
	// IP地理位置数据库配置（支持多个）
	IpLocateDbs []string
}

// DBEngine 存储所有数据库引擎实例（支持多数据库）
type DBEngine struct {
	// IP地理位置查询引擎（使用文件名作为键）
	Engines map[string]iplocate.IPInfo
}
