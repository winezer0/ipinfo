package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/winezer0/ipinfo/cmd/ipinfoapi/webapi"
	"github.com/winezer0/ipinfo/pkg/logging"
	"github.com/winezer0/ipinfo/pkg/queryip"
)

const (
	AppName    = "ipinfoapi"
	AppVersion = "1.0.0"
	BuildDate  = "2026-04-25"
)

func main() {
	config, err := LoadConfig(AppName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := initLogger(config); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync()

	logging.Infow("Starting application",
		"name", AppName,
		"version", AppVersion,
		"build_date", BuildDate,
	)

	if !config.HTTP.Enable {
		logging.Error("HTTP service is disabled")
		os.Exit(1)
	}

	engine, err := initDBEngine(config)
	if err != nil {
		logging.Errorw("Failed to initialize database engine", "error", err)
		os.Exit(1)
	}
	defer engine.Close()

	httpHandler := webapi.RegisterRoutes(engine, config.Auth.Token, config.Auth.Enable)

	srv := createHTTPServer(config, httpHandler)

	go func() {
		logging.Infow("Server starting",
			"addr", srv.Addr,
			"https", config.HTTP.HTTPS,
		)

		if config.HTTP.HTTPS {
			if err := srv.ListenAndServeTLS(config.HTTP.CertFile, config.HTTP.KeyFile); err != nil && err != http.ErrServerClosed {
				logging.Fatalw("HTTPS server failed", "error", err)
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logging.Fatalw("HTTP server failed", "error", err)
			}
		}
	}()

	waitForShutdown(srv)
}

// initLogger 初始化日志系统
func initLogger(config *ServerConfig) error {
	logCfg := logging.NewLogConfigWithRotation(
		config.Log.Level,
		config.Log.File,
		config.Log.Console,
		config.Log.MaxSize,
		config.Log.MaxBackups,
	)
	return logging.InitLogger(logCfg)
}

// initDBEngine 初始化数据库引擎
func initDBEngine(config *ServerConfig) (*queryip.DBEngine, error) {
	dbConfig := &queryip.IPDbConfig{
		AsnIpvxDb:    config.Database.AsnIpvxDb,
		AsnIpv4Db:    config.Database.AsnIpv4Db,
		AsnIpv6Db:    config.Database.AsnIpv6Db,
		IpvxLocateDb: config.Database.IpvxLocateDb,
		Ipv4LocateDb: config.Database.Ipv4LocateDb,
		Ipv6LocateDb: config.Database.Ipv6LocateDb,
	}

	engine, err := queryip.InitDBEngines(dbConfig)
	if err != nil {
		return nil, err
	}

	logging.Info("Database engines initialized")
	return engine, nil
}

// createHTTPServer 创建HTTP服务器，包含连接池配置
func createHTTPServer(config *ServerConfig, handler http.Handler) *http.Server {
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
	logging.Infow("Shutting down server", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logging.Errorw("Server forced to shutdown", "error", err)
	}

	logging.Info("Server stopped gracefully")
}
