package queryip

import (
	"errors"
	"sync"

	"github.com/winezer0/ipinfo/pkg/asninfo"
	"github.com/winezer0/ipinfo/pkg/iputils"
)

const (
	defaultWorkerPoolSize = 10
)

// QueryIPInfo 批量查询IP信息（使用worker pool模式）
func (engine *DBEngine) QueryIPInfo(ipv4s []string, ipv6s []string) (*IPDbInfo, error) {
	info := &IPDbInfo{
		IPv4Locations: make([]IPLocation, 0, len(ipv4s)),
		IPv6Locations: make([]IPLocation, 0, len(ipv6s)),
		IPv4AsnInfos:  make([]asninfo.ASNInfo, 0, len(ipv4s)),
		IPv6AsnInfos:  make([]asninfo.ASNInfo, 0, len(ipv6s)),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	if len(ipv4s) > 0 {
		ipv4WorkerPoolSize := defaultWorkerPoolSize
		if len(ipv4s) < ipv4WorkerPoolSize {
			ipv4WorkerPoolSize = len(ipv4s)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			ipv4Results := processIPsWithWorkerPool(ipv4s, engine.queryIPv4, ipv4WorkerPoolSize)

			mu.Lock()
			for _, result := range ipv4Results {
				if result == nil {
					continue
				}
				info.IPv4Locations = append(info.IPv4Locations, IPLocation{
					IP:       result.IP,
					IPLocate: result.IPLocate,
				})
				if result.ASNInfo != nil {
					info.IPv4AsnInfos = append(info.IPv4AsnInfos, *result.ASNInfo)
				}
			}
			mu.Unlock()
		}()
	}

	if len(ipv6s) > 0 {
		ipv6WorkerPoolSize := defaultWorkerPoolSize
		if len(ipv6s) < ipv6WorkerPoolSize {
			ipv6WorkerPoolSize = len(ipv6s)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			ipv6Results := processIPsWithWorkerPool(ipv6s, engine.queryIPv6, ipv6WorkerPoolSize)

			mu.Lock()
			for _, result := range ipv6Results {
				if result == nil {
					continue
				}
				info.IPv6Locations = append(info.IPv6Locations, IPLocation{
					IP:       result.IP,
					IPLocate: result.IPLocate,
				})
				if result.ASNInfo != nil {
					info.IPv6AsnInfos = append(info.IPv6AsnInfos, *result.ASNInfo)
				}
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	return info, nil
}

// processIPsWithWorkerPool 使用worker pool处理IP查询
func processIPsWithWorkerPool(ips []string, queryFunc func(string) *IPQueryResult, poolSize int) []*IPQueryResult {
	if len(ips) == 0 {
		return nil
	}

	results := make([]*IPQueryResult, len(ips))
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

// queryIPv4 查询单个IPv4地址
func (engine *DBEngine) queryIPv4(ip string) *IPQueryResult {
	result := &IPQueryResult{IP: ip}

	if engine.IPv4Engine != nil {
		result.IPLocate = engine.IPv4Engine.FindFull(ip)
	}

	if engine.AsnIPv4Engine != nil {
		result.ASNInfo = engine.AsnIPv4Engine.FindASN(ip)
	} else {
		result.ASNInfo = &asninfo.ASNInfo{IP: ip, IPVersion: 4, FoundASN: false}
	}

	return result
}

// queryIPv6 查询单个IPv6地址
func (engine *DBEngine) queryIPv6(ip string) *IPQueryResult {
	result := &IPQueryResult{IP: ip}

	if engine.IPv6Engine != nil {
		result.IPLocate = engine.IPv6Engine.FindFull(ip)
	}

	if engine.AsnIPv6Engine != nil {
		result.ASNInfo = engine.AsnIPv6Engine.FindASN(ip)
	} else {
		result.ASNInfo = &asninfo.ASNInfo{IP: ip, IPVersion: 6, FoundASN: false}
	}

	return result
}

// QueryIP 查询单个IP的信息
func (engine *DBEngine) QueryIP(ip string) *IPQueryResult {
	if iputils.IsIPv4(ip) {
		return engine.queryIPv4(ip)
	}
	return engine.queryIPv6(ip)
}

// Close 关闭所有数据库连接
func (engine *DBEngine) Close() error {
	var errs []error

	if engine.AsnIPv4Engine != nil {
		if err := engine.AsnIPv4Engine.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if engine.AsnIPv6Engine != nil && engine.AsnIPv6Engine != engine.AsnIPv4Engine {
		if err := engine.AsnIPv6Engine.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if engine.IPv4Engine != nil {
		engine.IPv4Engine.Close()
	}

	if engine.IPv6Engine != nil && engine.IPv6Engine != engine.IPv4Engine {
		engine.IPv6Engine.Close()
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
