package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/winezer0/ipinfo/pkg/iputils"
	"github.com/winezer0/ipinfo/pkg/queryip"

	"github.com/jessevdk/go-flags"
)

const (
	AppName      = "iplocate"
	AppVersion   = "1.0.0"
	BuildDate    = "2026-04-25"
	AppShortDesc = "IP Location Query Tool"
	AppLongDesc  = "A command-line tool for querying IP address location information using various database formats"
)

// CmdConfig 命令行参数配置
type CmdConfig struct {
	Version bool     `short:"v" long:"version" description:"显示版本信息并退出"`
	Config  string   `short:"c" long:"config" description:"指定配置文件路径"`
	IPs     []string `short:"I" long:"ip" description:"要查询的IP地址（可多次指定）"`
}

func main() {
	cmdConfig := parseArgs()

	if cmdConfig.Version {
		fmt.Printf("%s version %s\n", AppName, AppVersion)
		fmt.Printf("Build Date: %s\n", BuildDate)
		os.Exit(0)
	}

	if len(cmdConfig.IPs) == 0 {
		fmt.Fprintf(os.Stderr, "错误: 未提供IP地址，请使用 -I 指定IP\n")
		os.Exit(1)
	}

	configPath := cmdConfig.Config
	if configPath == "" {
		configPath = findConfigFile(AppName)
	}
	if configPath == "" {
		fmt.Fprintf(os.Stderr, "错误: 未找到配置文件\n")
		os.Exit(1)
	}

	config, err := LoadConfigFromFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 加载配置文件失败: %v\n", err)
		os.Exit(1)
	}

	dbConfig := &queryip.IPDbConfig{
		AsnIpvxDb:    config.AsnIpvxDb,
		AsnIpv4Db:    config.AsnIpv4Db,
		AsnIpv6Db:    config.AsnIpv6Db,
		IpvxLocateDb: config.IpvxLocateDb,
		Ipv4LocateDb: config.Ipv4LocateDb,
		Ipv6LocateDb: config.Ipv6LocateDb,
	}

	engine, err := queryip.InitDBEngines(dbConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 初始化数据库引擎失败: %v\n", err)
		os.Exit(1)
	}
	defer engine.Close()

	ipClass := iputils.ClassifyIPs(cmdConfig.IPs)

	results, err := engine.QueryIPInfo(ipClass.IPv4s, ipClass.IPv6s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 查询IP信息失败: %v\n", err)
		os.Exit(1)
	}

	outputJSON(results)
}

// parseArgs 解析命令行参数
func parseArgs() *CmdConfig {
	var cmdConfig = &CmdConfig{}
	parser := flags.NewParser(cmdConfig, flags.Default^flags.PassDoubleDash)
	parser.Name = AppName
	parser.Usage = "[OPTIONS]"
	parser.ShortDescription = AppShortDesc
	parser.LongDescription = AppLongDesc

	_, err := parser.Parse()
	if err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	return cmdConfig
}

// outputJSON 将查询结果以JSON格式输出到控制台
func outputJSON(results *queryip.IPDbInfo) {
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: JSON序列化失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}
