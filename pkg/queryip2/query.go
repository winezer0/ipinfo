package queryip2

import (
	"sync"

	"github.com/winezer0/ipinfo/pkg/iplocate"
	"github.com/winezer0/ipinfo/pkg/iputils"
)

const (
	defaultWorkerPoolSize = 10
)

// QueryIPInfo 批量查询IP信息（使用多数据库并行查询）
func (engine *DBEngine) QueryIPInfo(ipv4s []string, ipv6s []string) (*IPDbInfo, error) {
	info := &IPDbInfo{
		IPv4Results: make([]IPQueryResult, 0, len(ipv4s)),
		IPv6Results: make([]IPQueryResult, 0, len(ipv6s)),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	workerPoolSize := defaultWorkerPoolSize
	if len(ipv4s) < workerPoolSize {
		workerPoolSize = len(ipv4s)
	}
	if len(ipv6s) < workerPoolSize && len(ipv6s) > 0 {
		workerPoolSize = len(ipv6s)
	}

	if len(ipv4s) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ipv4Results := processIPsWithWorkerPool(ipv4s, engine.queryIP, workerPoolSize)

			mu.Lock()
			info.IPv4Results = append(info.IPv4Results, ipv4Results...)
			mu.Unlock()
		}()
	}

	if len(ipv6s) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ipv6Results := processIPsWithWorkerPool(ipv6s, engine.queryIP, workerPoolSize)

			mu.Lock()
			info.IPv6Results = append(info.IPv6Results, ipv6Results...)
			mu.Unlock()
		}()
	}

	wg.Wait()
	return info, nil
}

// QueryIPInfoByIPs 根据IP列表批量查询（自动分类IPv4/IPv6）
func (engine *DBEngine) QueryIPInfoByIPs(ips []string) (*IPDbInfo, error) {
	ipClass := iputils.ClassifyIPs(ips)
	return engine.QueryIPInfo(ipClass.IPv4s, ipClass.IPv6s)
}

// processIPsWithWorkerPool 使用worker pool处理IP查询
func processIPsWithWorkerPool(ips []string, queryFunc func(string) IPQueryResult, poolSize int) []IPQueryResult {
	if len(ips) == 0 {
		return nil
	}

	results := make([]IPQueryResult, len(ips))
	jobChan := make(chan int, len(ips))
	var wg sync.WaitGroup

	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobChan {
				results[idx] = queryFunc(ips[idx])
			}
		}()
	}

	for i := range ips {
		jobChan <- i
	}
	close(jobChan)

	wg.Wait()
	return results
}

// queryIP 查询单个IP地址（根据IP版本和引擎支持情况动态调用）
func (engine *DBEngine) queryIP(ip string) IPQueryResult {
	result := IPQueryResult{
		IP:             ip,
		IPLocateResult: make(map[Source]*iplocate.IPLocate),
	}

	ipVersion := iputils.GetIpVersion(ip)
	isIPv4 := ipVersion == 4

	var wg sync.WaitGroup
	var mu sync.Mutex

	for source, dbEngine := range engine.Engines {
		dbInfo := dbEngine.GetDatabaseInfo()

		// 根据IP版本判断引擎是否支持
		if isIPv4 && !dbInfo.IsIPv4 {
			continue
		}
		if !isIPv4 && !dbInfo.IsIPv6 {
			continue
		}

		wg.Add(1)
		go func(src string, eng iplocate.IPInfo) {
			defer wg.Done()
			locateResult := eng.FindFull(ip)
			if locateResult != nil {
				mu.Lock()
				result.IPLocateResult[Source(src)] = locateResult
				mu.Unlock()
			}
		}(source, dbEngine)
	}

	wg.Wait()
	return result
}

// QueryIP 查询单个IP的信息（使用所有匹配的数据库）
func (engine *DBEngine) QueryIP(ip string) IPQueryResult {
	return engine.queryIP(ip)
}

// Close 关闭所有数据库连接
func (engine *DBEngine) Close() error {
	closeIPEngines(engine.Engines)
	return nil
}
