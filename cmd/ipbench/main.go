// ipbench measures XDB query latency and throughput with bounded independent searchers.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/winezer0/ipinfo/internal/benchutil"
	"github.com/winezer0/ipinfo/pkg/iplocate/ip2region"
)

type result struct {
	Pool              int     `json:"pool"`
	Workers           int     `json:"workers"`
	Batch             int     `json:"batch"`
	Requests          int     `json:"requests"`
	IPs               int     `json:"ips"`
	Misses            int64   `json:"misses"`
	Seconds           float64 `json:"seconds"`
	RequestsPerSecond float64 `json:"requests_per_second"`
	IPsPerSecond      float64 `json:"ips_per_second"`
	P50Microseconds   float64 `json:"p50_us"`
	P95Microseconds   float64 `json:"p95_us"`
	BytesPerRequest   float64 `json:"bytes_per_request"`
	AllocsPerRequest  float64 `json:"allocs_per_request"`
	PoolHeapBytes     int64   `json:"pool_heap_bytes"`
}

type report struct {
	Kind           string            `json:"kind"`
	Implementation string            `json:"implementation"`
	Dataset        map[string]string `json:"dataset"`
	Go             string            `json:"go"`
	OS             string            `json:"os"`
	Arch           string            `json:"arch"`
	CPUs           int               `json:"cpus"`
	GOMAXPROCS     int               `json:"gomaxprocs"`
	Database       string            `json:"database"`
	DatabaseBytes  int64             `json:"database_bytes"`
	Version        int               `json:"version"`
	Clock          string            `json:"clock"`
	Note           string            `json:"note"`
	Results        []result          `json:"results"`
}

func numbers(value string) ([]int, error) {
	var values []int
	for _, part := range strings.Split(value, ",") {
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > 10000 {
			return nil, fmt.Errorf("invalid positive count: %q", part)
		}
		values = append(values, n)
	}
	return values, nil
}

// run measures the production XDB pool, including lease wait time.
func run(path string, version, poolSize, workers, batch, requests int) (result, error) {
	runtime.GC()
	var beforePool, afterPool runtime.MemStats
	runtime.ReadMemStats(&beforePool)
	db, err := ip2region.NewIP2RegionWithPool(version, path, poolSize)
	if err != nil {
		return result{}, err
	}
	defer db.Close()
	base := []string{"1.1.1.1", "8.8.8.8", "114.114.114.114", "223.5.5.5", "9.9.9.9", "208.67.222.222", "8.8.4.4", "180.76.76.76"}
	if version == 6 {
		base = []string{"2001:4860:4860::8888", "2606:4700:4700::1111", "2400:3200::1", "2402:4e00::", "2001:4860:4860::8844", "2606:4700:4700::1001"}
	}
	queries := make([][]string, 32)
	for i := range queries {
		queries[i] = make([]string, batch)
		for j := range queries[i] {
			queries[i][j] = base[(i+j)%len(base)]
		}
	}
	for _, ip := range base {
		if !db.FindFull(ip).HasData() {
			return result{}, fmt.Errorf("warmup miss: %s", ip)
		}
	}
	runtime.GC()
	runtime.ReadMemStats(&afterPool)
	latencies := make([]time.Duration, requests)
	var next atomic.Int64
	var misses atomic.Int64
	var wg sync.WaitGroup
	ready, start := sync.WaitGroup{}, make(chan struct{})
	ready.Add(workers)
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ready.Done()
			<-start
			for {
				i := int(next.Add(1) - 1)
				if i >= requests {
					return
				}
				q := queries[i%len(queries)]
				t := benchutil.Tick()
				if batch == 1 {
					if !db.FindFull(q[0]).HasData() {
						misses.Add(1)
					}
				} else {
					for _, item := range db.BatchFindFull(q) {
						if !item.HasData() {
							misses.Add(1)
						}
					}
				}
				latencies[i] = benchutil.Since(t)
			}
		}()
	}
	ready.Wait()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	t := benchutil.Tick()
	close(start)
	wg.Wait()
	elapsed := benchutil.Since(t)
	runtime.ReadMemStats(&after)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	return result{Pool: poolSize, Workers: workers, Batch: batch, Requests: requests, IPs: requests * batch, Misses: misses.Load(), Seconds: elapsed.Seconds(), RequestsPerSecond: float64(requests) / elapsed.Seconds(), IPsPerSecond: float64(requests*batch) / elapsed.Seconds(), P50Microseconds: float64(latencies[(requests-1)*50/100]) / 1000, P95Microseconds: float64(latencies[(requests-1)*95/100]) / 1000, BytesPerRequest: float64(after.TotalAlloc-before.TotalAlloc) / float64(requests), AllocsPerRequest: float64(after.Mallocs-before.Mallocs) / float64(requests), PoolHeapBytes: int64(afterPool.HeapAlloc) - int64(beforePool.HeapAlloc)}, nil
}

func main() {
	path := flag.String("xdb", "", "required XDB path")
	workerArg := flag.String("workers", "1,4,16", "concurrent workers")
	poolArg := flag.String("pool", "1,4", "independent searchers, sharing VectorIndex")
	batchArg := flag.String("batch", "1,16", "IPs per request; repeated IPs still execute lookups")
	requests := flag.Int("requests", 10000, "requests per scenario")
	out := flag.String("out", "", "JSON output file (default stdout)")
	flag.Parse()
	if err := execute(*path, *workerArg, *poolArg, *batchArg, *requests, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute(path, workerArg, poolArg, batchArg string, requests int, out string) error {
	if requests < 1 {
		return fmt.Errorf("requests must be positive")
	}
	workers, err := numbers(workerArg)
	if err != nil {
		return err
	}
	pools, err := numbers(poolArg)
	if err != nil {
		return err
	}
	batches, err := numbers(batchArg)
	if err != nil {
		return err
	}
	db, err := ip2region.NewIP2RegionAuto(path)
	if err != nil {
		return err
	}
	version := db.Version
	db.Close()
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	r := report{Go: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH, CPUs: runtime.NumCPU(), GOMAXPROCS: runtime.GOMAXPROCS(0), Database: path, DatabaseBytes: info.Size(), Version: version, Clock: benchutil.ClockName, Note: "Closed-loop XDB benchmark; warm OS cache; latency includes pool wait; excludes HTTP/ASN/network and startup. Heap delta is approximate, not RSS. Batch IPs may repeat. Run multiple times on an otherwise idle machine."}
	hash, err := benchutil.FileSHA256(path)
	if err != nil {
		return err
	}
	r.Kind = "xdb"
	r.Implementation = "production pool"
	r.Dataset = map[string]string{fmt.Sprintf("ipv%d", version): hash}
	for _, pool := range pools {
		for _, batch := range batches {
			for _, worker := range workers {
				fmt.Fprintf(os.Stderr, "pool=%d batch=%d workers=%d\n", pool, batch, worker)
				item, err := run(path, version, pool, worker, batch, requests)
				if err != nil {
					return err
				}
				r.Results = append(r.Results, item)
			}
		}
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if out != "" {
		return os.WriteFile(out, data, 0644)
	}
	_, err = os.Stdout.Write(data)
	return err
}
