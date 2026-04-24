package iplocate

// IPInfo 接口定义了IP地址信息查询的标准方法
type IPInfo interface {
	// Init 初始化数据库连接
	Init() error

	// IsInitialized 检查数据库是否已初始化
	IsInitialized() bool

	// FindFull 查询单个IP地址的地理位置信息,返回更结构化的地址信息
	FindFull(query string) *IPLocate

	// BatchFindFull 批量查询多个IP地址,返回更结构化的地址信息
	BatchFindFull(queries []string) map[string]*IPLocate

	// GetDatabaseInfo 获取数据库信息
	GetDatabaseInfo() *DBInfo

	// Close 关闭数据库连接（清理资源）
	Close()
}
