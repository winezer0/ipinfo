// apibench benchmarks the complete API over loopback HTTP in a separate process.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/winezer0/ipinfo/cmd/ipinfoapi/webapi"
	"github.com/winezer0/ipinfo/internal/benchutil"
	"github.com/winezer0/ipinfo/pkg/queryip"
	"github.com/winezer0/slogs"
)

const token = "local-apibench-only"
const metricsPath = "/__apibench_usage"

var sampleIPs = []string{"8.8.8.8", "1.1.1.1", "114.114.114.114", "9.9.9.9", "2001:4860:4860::8888", "2606:4700:4700::1111", "2400:3200::1", "2001:4860:4860::8844"}

type measurement struct {
	Repeat                    int     `json:"repeat"`
	Pool                      int     `json:"pool"`
	Workers                   int     `json:"workers"`
	Endpoint                  string  `json:"endpoint"`
	Batch                     int     `json:"batch"`
	Requests                  int     `json:"requests"`
	Errors                    int64   `json:"errors"`
	ErrorRate                 float64 `json:"error_rate"`
	FirstError                string  `json:"first_error,omitempty"`
	Seconds                   float64 `json:"seconds"`
	RequestsPerSecond         float64 `json:"requests_per_second"`
	P50US                     float64 `json:"p50_us"`
	P95US                     float64 `json:"p95_us"`
	ServerCPUSeconds          float64 `json:"server_cpu_seconds"`
	ServerCPUPercent          float64 `json:"server_cpu_percent"`
	ServerRSSMaxSampledBytes  uint64  `json:"server_rss_max_sampled_bytes"`
	ServerProcessPeakRSSBytes uint64  `json:"server_process_peak_rss_bytes"`
}

type report struct {
	Dataset    map[string]string `json:"dataset"`
	Kind       string            `json:"kind"`
	Go         string            `json:"go"`
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	CPUs       int               `json:"cpus"`
	GOMAXPROCS int               `json:"gomaxprocs"`
	Clock      string            `json:"clock"`
	Assets     string            `json:"assets"`
	Note       string            `json:"note"`
	Results    []measurement     `json:"results"`
}

func serve(assets string, pool int) error {
	if err := slogs.Init(slogs.NewConfig("error", "", "LM")); err != nil {
		return err
	}
	db, err := queryip.InitDBEngines(&queryip.IPDbConfig{XDBPoolSize: pool, AsnIpvxDb: filepath.Join(assets, "geolite2-asn.mmdb"), Ipv4LocateDb: filepath.Join(assets, "ip2region_v4.xdb"), Ipv6LocateDb: filepath.Join(assets, "ip2region_v6.xdb")})
	if err != nil {
		return err
	}
	defer db.Close()
	handler := webapi.RegisterRoutes(db, token, true)
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == metricsPath {
			if r.Header.Get("Authorization") != "Bearer "+token {
				w.WriteHeader(401)
				return
			}
			usage, err := benchutil.ProcessUsage()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			json.NewEncoder(w).Encode(usage)
			return
		}
		handler.ServeHTTP(w, r)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	server := &http.Server{Handler: wrapped, ReadHeaderTimeout: 5 * time.Second}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]string{"url": "http://" + listener.Addr().String()}); err != nil {
		listener.Close()
		return err
	}
	go func() {
		io.Copy(io.Discard, os.Stdin)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()
	err = server.Serve(listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

type child struct {
	cmd   *exec.Cmd
	input io.WriteCloser
	url   string
}

func (c *child) close() {
	c.input.Close()
	done := make(chan struct{})
	go func() { c.cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(7 * time.Second):
		c.cmd.Process.Kill()
		<-done
	}
}

func startChild(assets string, pool int) (*child, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(exe, "-child", "-assets", assets, "-pool", strconv.Itoa(pool))
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		input.Close()
		output.Close()
		return nil, err
	}
	c := &child{cmd: cmd, input: input}
	type ready struct {
		url string
		err error
	}
	channel := make(chan ready, 1)
	go func() {
		var data struct {
			URL string `json:"url"`
		}
		err := json.NewDecoder(output).Decode(&data)
		channel <- ready{data.URL, err}
	}()
	select {
	case data := <-channel:
		if data.err != nil || data.url == "" {
			c.close()
			return nil, fmt.Errorf("child startup: %v", data.err)
		}
		c.url = data.url
		return c, nil
	case <-time.After(30 * time.Second):
		c.close()
		return nil, fmt.Errorf("child startup timeout")
	}
}

func request(client *http.Client, base, path string) ([]byte, int, error) {
	r, err := http.NewRequest(http.MethodGet, base+path, nil)
	if err != nil {
		return nil, 0, err
	}
	r.Header.Set("Authorization", "Bearer "+token)
	response, err := client.Do(r)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	return data, response.StatusCode, err
}

func usage(client *http.Client, base string) (benchutil.Usage, error) {
	data, status, err := request(client, base, metricsPath)
	if err != nil {
		return benchutil.Usage{}, err
	}
	if status != 200 {
		return benchutil.Usage{}, fmt.Errorf("usage status %d", status)
	}
	var u benchutil.Usage
	err = json.Unmarshal(data, &u)
	return u, err
}

func validate(data []byte, status int, endpoint string, expected []string) error {
	var envelope struct {
		Success bool            `json:"success"`
		Code    int             `json:"code"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	if status != 200 || !envelope.Success || envelope.Code != 0 {
		return fmt.Errorf("API failure status=%d code=%d", status, envelope.Code)
	}
	switch endpoint {
	case "/ip":
		var item webapi.IPLocateData
		if err := json.Unmarshal(envelope.Data, &item); err != nil {
			return err
		}
		if item.IP != expected[0] || item.Location == "" {
			return fmt.Errorf("location missing or wrong IP")
		}
	case "/asn":
		var item webapi.ASNData
		if err := json.Unmarshal(envelope.Data, &item); err != nil {
			return err
		}
		if item.IP != expected[0] || !item.FoundASN || item.ASNumber == 0 {
			return fmt.Errorf("ASN missing or wrong IP")
		}
	case "/all":
		var item webapi.AllData
		if err := json.Unmarshal(envelope.Data, &item); err != nil {
			return err
		}
		if item.IP != expected[0] || item.Location == nil || item.Location.Location == "" || item.ASN == nil || !item.ASN.FoundASN {
			return fmt.Errorf("incomplete combined result")
		}
	case "/full":
		var item webapi.FullData
		if err := json.Unmarshal(envelope.Data, &item); err != nil {
			return err
		}
		if item.Total != len(expected) || item.Success != len(expected) || item.Failed != 0 || len(item.Results) != len(expected) {
			return fmt.Errorf("batch counts mismatch")
		}
		for i, result := range item.Results {
			if result.IP != expected[i] || result.Error != "" || result.Location == nil || result.Location.Location == "" || result.ASN == nil || !result.ASN.FoundASN {
				return fmt.Errorf("incomplete batch item %s", result.IP)
			}
		}
	default:
		return fmt.Errorf("unknown endpoint %s", endpoint)
	}
	return nil
}

func measure(base, endpoint string, workers, requests, batch int) (measurement, error) {
	transport := &http.Transport{MaxIdleConns: workers + 2, MaxIdleConnsPerHost: workers + 2, MaxConnsPerHost: workers + 2}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}
	paths := make([]string, len(sampleIPs))
	expected := make([][]string, len(sampleIPs))
	for i, ip := range sampleIPs {
		expected[i] = []string{ip}
		if endpoint == "/full" {
			expected[i] = sampleIPs[:batch]
		}
		paths[i] = endpoint + "?q=" + url.QueryEscape(strings.Join(expected[i], ","))
	}
	query := func(index int) error {
		data, status, err := request(client, base, paths[index%len(paths)])
		if err != nil {
			return err
		}
		return validate(data, status, endpoint, expected[index%len(paths)])
	}
	for i := 0; i < 32; i++ {
		if err := query(i); err != nil {
			return measurement{}, fmt.Errorf("warmup %s: %w", endpoint, err)
		}
	}
	before, err := usage(client, base)
	if err != nil {
		return measurement{}, err
	}
	var maxRSS atomic.Uint64
	maxRSS.Store(before.RSSBytes)
	var samplingError error
	stop, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		timer := time.NewTicker(20 * time.Millisecond)
		defer timer.Stop()
		for {
			select {
			case <-stop:
				return
			case <-timer.C:
				u, err := usage(client, base)
				if err != nil {
					samplingError = err
					return
				}
				if u.RSSBytes > maxRSS.Load() {
					maxRSS.Store(u.RSSBytes)
				}
			}
		}
	}()
	latencies := make([]time.Duration, requests)
	var next, errors atomic.Int64
	var firstError string
	var errorOnce sync.Once
	var wg, ready sync.WaitGroup
	start := make(chan struct{})
	ready.Add(workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ready.Done()
			<-start
			for {
				n := int(next.Add(1) - 1)
				if n >= requests {
					return
				}
				t := benchutil.Tick()
				err := query(n)
				latencies[n] = benchutil.Since(t)
				if err != nil {
					errors.Add(1)
					errorOnce.Do(func() { firstError = err.Error() })
				}
			}
		}()
	}
	ready.Wait()
	t := benchutil.Tick()
	close(start)
	wg.Wait()
	elapsed := benchutil.Since(t)
	close(stop)
	<-stopped
	if samplingError != nil {
		return measurement{}, samplingError
	}
	after, err := usage(client, base)
	if err != nil {
		return measurement{}, err
	}
	if after.RSSBytes > maxRSS.Load() {
		maxRSS.Store(after.RSSBytes)
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	if endpoint != "/full" {
		batch = 1
	}
	cpu := after.CPUSeconds - before.CPUSeconds
	return measurement{Workers: workers, Endpoint: endpoint, Batch: batch, Requests: requests, Errors: errors.Load(), ErrorRate: float64(errors.Load()) / float64(requests), FirstError: firstError, Seconds: elapsed.Seconds(), RequestsPerSecond: float64(requests) / elapsed.Seconds(), P50US: float64(latencies[(requests-1)/2]) / 1000, P95US: float64(latencies[(requests-1)*95/100]) / 1000, ServerCPUSeconds: cpu, ServerCPUPercent: cpu / elapsed.Seconds() * 100, ServerRSSMaxSampledBytes: maxRSS.Load(), ServerProcessPeakRSSBytes: after.PeakRSSBytes}, nil
}

func positiveList(value string, max int) ([]int, error) {
	var result []int
	seen := map[int]bool{}
	for _, part := range strings.Split(value, ",") {
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > max || seen[n] {
			return nil, fmt.Errorf("invalid counts: %s", value)
		}
		seen[n] = true
		result = append(result, n)
	}
	return result, nil
}

func execute(assets, poolArg, workerArg, out string, requests, repeats, batch int) error {
	if requests < 1 || repeats < 1 || batch < 1 || batch > len(sampleIPs) {
		return fmt.Errorf("invalid requests/repeats/batch")
	}
	pools, err := positiveList(poolArg, 64)
	if err != nil {
		return err
	}
	workers, err := positiveList(workerArg, 512)
	if err != nil {
		return err
	}
	assets, err = filepath.Abs(assets)
	if err != nil {
		return err
	}
	r := report{Kind: "api", Go: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH, CPUs: runtime.NumCPU(), GOMAXPROCS: runtime.GOMAXPROCS(0), Clock: benchutil.ClockName, Assets: assets, Note: "Separate server process; loopback HTTP/1.1 keep-alive; real v4/v6 XDB+ASN; auth, recovery, client-IP and logging middleware enabled, log level error. Closed-loop latency includes client JSON validation. Server CPU is percent of one core (may exceed 100); RSS sampled every 20ms; process peak RSS is lifetime. Metrics requests add small overhead. No TLS, WAN or disk log writes."}
	var failures int64
	r.Dataset = map[string]string{}
	for _, name := range []string{"geolite2-asn.mmdb", "ip2region_v4.xdb", "ip2region_v6.xdb"} {
		hash, err := benchutil.FileSHA256(filepath.Join(assets, name))
		if err != nil {
			return err
		}
		r.Dataset[name] = hash
	}
	for repeat := 1; repeat <= repeats; repeat++ {
		for _, pool := range pools {
			err := func() error {
				server, err := startChild(assets, pool)
				if err != nil {
					return err
				}
				defer server.close()
				for _, worker := range workers {
					for _, endpoint := range []string{"/ip", "/asn", "/all", "/full"} {
						fmt.Fprintf(os.Stderr, "repeat=%d pool=%d workers=%d endpoint=%s\n", repeat, pool, worker, endpoint)
						item, err := measure(server.url, endpoint, worker, requests, batch)
						if err != nil {
							return err
						}
						item.Repeat = repeat
						item.Pool = pool
						r.Results = append(r.Results, item)
						failures += item.Errors
					}
				}
				return nil
			}()
			if err != nil {
				return err
			}
		}
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if out != "" {
		err = os.WriteFile(out, data, 0644)
	} else {
		_, err = os.Stdout.Write(data)
	}
	if err != nil {
		return err
	}
	if failures > 0 {
		return fmt.Errorf("%d measured API requests failed; see report", failures)
	}
	return nil
}

func main() {
	childMode := flag.Bool("child", false, "internal benchmark server mode (loopback only)")
	assets := flag.String("assets", "assets", "database directory")
	pools := flag.String("pool", "1,4", "pool sizes")
	workers := flag.String("workers", "1,4,16", "concurrency levels")
	requests := flag.Int("requests", 2000, "requests per scenario")
	repeats := flag.Int("repeat", 3, "number of repetitions")
	batch := flag.Int("batch", 8, "distinct IPs in /full, 1–8")
	out := flag.String("out", "", "JSON result path")
	flag.Parse()
	var err error
	if *childMode {
		var sizes []int
		sizes, err = positiveList(*pools, 64)
		if err == nil {
			if len(sizes) != 1 {
				err = fmt.Errorf("child requires one pool size")
			} else {
				err = serve(*assets, sizes[0])
			}
		}
	} else {
		err = execute(*assets, *pools, *workers, *out, *requests, *repeats, *batch)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
