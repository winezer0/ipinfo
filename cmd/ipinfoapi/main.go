package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/winezer0/ipinfo/cmd/ipinfoapi/config"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/winezer0/downutils/downutils"
	"github.com/winezer0/ipinfo/cmd/ipinfoapi/webapi"
	"github.com/winezer0/ipinfo/pkg/queryip"
	"github.com/winezer0/xutils/logging"
)

const (
	AppName      = "ipinfoapi"
	AppVersion   = "0.0.1"
	BuildDate    = "2026-04-27"
	AppShortDesc = "ip info web api"
	AppLongDesc  = "ip info web api"
)

// Options command line options
type Options struct {
	ConfigPath     string `short:"c" long:"config" description:"custom yaml config file path"`
	GenerateConfig bool   `long:"gen" description:"gen default config to <ConfigPath>"`
	Version        bool   `short:"v" long:"version" description:"output version information"`
}

// InitOptionsArgs 常用的工具函数，解析parser和logging配置
func InitOptionsArgs(minimumParams int) (*Options, *flags.Parser) {
	opts := &Options{}
	parser := flags.NewParser(opts, flags.Default)
	parser.Name = AppName
	parser.Usage = "[OPTIONS]"
	parser.ShortDescription = AppShortDesc
	parser.LongDescription = AppLongDesc

	// 命令行参数数量检查 指不包含程序名本身的参数数量
	if minimumParams > 0 && len(os.Args)-1 < minimumParams {
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	// 命令行参数解析检查
	if _, err := parser.Parse(); err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && errors.Is(flagsErr.Type, flags.ErrHelp) {
			os.Exit(0)
		}
		fmt.Printf("Error:%v\n", err)
		os.Exit(1)
	}

	// 版本号输出
	if opts.Version {
		fmt.Printf("%s version %s\n", AppName, AppVersion)
		fmt.Printf("Build Date: %s\n", BuildDate)
		os.Exit(0)
	}

	// 处理生成配置文件命令
	if opts.GenerateConfig {
		configPath := opts.ConfigPath
		if configPath == "" {
			configPath = AppName + ".yaml"
		}
		if err := config.GenDefaultConfig(configPath); err != nil {
			fmt.Printf("Failed to generate config file: %v\n", err)
		}
		fmt.Printf("Default config file has been generated: %s\n", configPath)
		os.Exit(0)
	}

	return opts, parser
}

func main() {
	// 打印命令行输入配置
	opts, _ := InitOptionsArgs(0)

	cfg, err := config.LoadConfig(opts.ConfigPath, AppName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logCfg := logging.LogConfig{
		Level:         cfg.Log.Level,
		LogFile:       cfg.Log.File,
		ConsoleFormat: cfg.Log.Console,
		MaxSize:       cfg.Log.MaxSize,    // 单个文件最大100MB
		MaxBackups:    cfg.Log.MaxBackups, // 最多保留10个备份
		MaxAge:        30,                 // 保留30天
		Compress:      true,               // 压缩备份文件
	}

	if err := logging.InitDefaultLogger(logCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init default logger: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync()

	logging.Infof("Starting application name: %s %s %s", AppName, AppVersion, BuildDate)

	if err := downloadDatabases(cfg); err != nil {
		logging.Warnf("Database download failed, continuing with existing databases error: %v", err)
	}

	if !cfg.HTTP.Enable {
		logging.Error("HTTP service is disabled")
		os.Exit(1)
	}

	engine, err := initDBEngine(cfg)
	if err != nil {
		logging.Errorf("Failed to initialize database engineerror :%v", err)
		os.Exit(1)
	}
	defer engine.Close()

	httpHandler := webapi.RegisterRoutes(engine, cfg.Auth.Token, cfg.Auth.Enable)

	srv := createHTTPServer(cfg, httpHandler)

	go func() {
		logging.Infof("Server starting srv.Addr:%v EnableHttps(%v)", srv.Addr, cfg.HTTP.HTTPS)

		if cfg.HTTP.HTTPS {
			if err := srv.ListenAndServeTLS(cfg.HTTP.CertFile, cfg.HTTP.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logging.Fatalf("HTTPS server failed:%v", err)
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logging.Fatalf("HTTP server failed:%v", err)
			}
		}
	}()

	waitForShutdown(srv)
}

// downloadDatabases 下载IP数据库
func downloadDatabases(config *config.ServerConfig) error {
	if len(config.Databases) == 0 {
		return nil
	}

	downConfig := downutils.DownConfig{
		"databases": config.Databases,
	}

	downOptions := &downutils.DownOptions{
		OutputForce:    "",
		UpdateForce:    false,
		EnableForce:    false,
		ShowProgress:   true,
		MaxRetries:     3,
		MaxConcurrent:  3,
		MaxSpeed:       0,
		ConnectTimeout: 30,
		IdleTimeout:    120,
		ProxyURL:       "",
	}

	result, err := downutils.ExecuteDownloads(downOptions, downConfig)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	downutils.DisplayDownloadResult(result)

	if result.FailedItems > 0 {
		return fmt.Errorf("%d database downloads failed", result.FailedItems)
	}

	return nil
}

// initDBEngine 初始化数据库引擎
func initDBEngine(config *config.ServerConfig) (*queryip.DBEngine, error) {
	dbConfig := &queryip.IPDbConfig{
		AsnIpvxDb:    downutils.GetModuleFinalPath("asn_ipvx_db", config.Databases, ""),
		AsnIpv4Db:    downutils.GetModuleFinalPath("asn_ipv4_db", config.Databases, ""),
		AsnIpv6Db:    downutils.GetModuleFinalPath("asn_ipv6_db", config.Databases, ""),
		IpvxLocateDb: downutils.GetModuleFinalPath("ipvx_locate_db", config.Databases, ""),
		Ipv4LocateDb: downutils.GetModuleFinalPath("ipv4_locate_db", config.Databases, ""),
		Ipv6LocateDb: downutils.GetModuleFinalPath("ipv6_locate_db", config.Databases, ""),
	}

	engine, err := queryip.InitDBEngines(dbConfig)
	if err != nil {
		return nil, err
	}

	logging.Info("Database engines initialized")
	return engine, nil
}

// createHTTPServer 创建HTTP服务器，包含连接池配置
func createHTTPServer(config *config.ServerConfig, handler http.Handler) *http.Server {
	addr := fmt.Sprintf(":%d", config.HTTP.Port)

	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  time.Duration(config.HTTP.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.HTTP.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(config.HTTP.IdleTimeout) * time.Second,
	}
}

// waitForShutdown 等待关闭信号并优雅停止服务器
func waitForShutdown(srv *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	logging.Infof("Shutting down server signal: %v", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logging.Errorf("Server forced to shutdown error: %v", err)
	}

	logging.Info("Server stopped gracefully")
}
