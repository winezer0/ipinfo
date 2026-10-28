# ipinfo

IP 地理位置与 ASN 信息查询工具集，支持多种数据库格式，提供 Go 包调用和 HTTP API 服务。

## 支持的数据库格式

### IP 地理位置数据库

| 格式                           | 说明            | 支持版本      |
| ---------------------------- | ------------- | --------- |
| qqwry.dat                    | 纯真 IP 数据库     | IPv4      |
| zxipv6wry.db                 | ZX IPv6 数据库   | IPv6      |
| ip2region\_v4/v6.xdb         | IP2Region     | IPv4/IPv6 |
| geolite2-city-ipv4/ipv6.mmdb | GeoLite2 City | IPv4/IPv6 |
| dbip-city-ipv4/ipv6.mmdb     | DBIP City     | IPv4/IPv6 |
| qqwry.ipdb                   | IPIP.net 格式   | IPv4      |

### ASN 数据库

| 格式                          | 说明           | 支持版本      |
| --------------------------- | ------------ | --------- |
| geolite2-asn.mmdb           | GeoLite2 ASN | IPv4/IPv6 |
| dbip-asn.mmdb               | DBIP ASN     | IPv4/IPv6 |
| geolite2/dbip-asn-ipv4.mmdb | 独立 IPv4 库    | IPv4      |
| geolite2/dbip-asn-ipv6.mmdb | 独立 IPv6 库    | IPv6      |
| dbip-asn-ipv4/ipv6.csv      | CSV 格式       | IPv4/IPv6 |

***

## 项目结构

```
ipinfo/
├── pkg/                     # 核心功能包
│   ├── iplocate/            # IP 地理位置查询接口与实现
│   ├── asninfo/             # ASN 信息查询接口与实现
│   ├── queryip/             # 统一查询引擎（组合 IP + ASN）
│   ├── multiloc/            # 多数据库并行定位查询引擎
│   ├── iputils/             # IP 工具函数
│   └── utils/               # 通用工具
├── cmd/                     # 可执行程序
│   ├── iplocate/            # 命令行 IP 查询工具
│   ├── ipmultiloc/          # 命令行工具（多数据库版本）
│   └── ipinfoapi/           # HTTP API 服务
└── assets/                  # 数据库文件目录
```

***

## pkg 包使用指南

### 1. pkg/queryip - 统一查询引擎

核心包，组合 IP 地理位置和 ASN 查询功能。

#### 初始化引擎

```go
package main

import (
    "fmt"
    "github.com/winezer0/ipinfo/pkg/queryip"
)

func main() {
    config := &queryip.IPDbConfig{
        AsnIpvxDb:    "assets/geolite2-asn.mmdb",
        Ipv4LocateDb: "assets/qqwry.dat",
        Ipv6LocateDb: "assets/zxipv6wry.db",
    }

    engine, err := queryip.InitDBEngines(config)
    if err != nil {
        panic(err)
    }
    defer engine.Close()

    // 查询单个 IP
    result := engine.QueryIP("8.8.8.8")
    fmt.Printf("IP: %s\n", result.IP)
    if result.IPLocate != nil {
        fmt.Printf("位置: %s %s %s\n", result.IPLocate.Country, result.IPLocate.Province, result.IPLocate.City)
    }
    if result.ASNInfo != nil {
        fmt.Printf("ASN: %d %s\n", result.ASNInfo.OrganisationNumber, result.ASNInfo.OrganisationName)
    }
}
```

#### 批量查询

```go
// 批量查询 IPv4 和 IPv6
ipv4s := []string{"8.8.8.8", "1.1.1.1", "114.114.114.114"}
ipv6s := []string{"2001:4860:4860::8888"}

info, err := engine.QueryIPInfo(ipv4s, ipv6s)
if err != nil {
    panic(err)
}

// 遍历结果
for _, loc := range info.IPv4Locations {
    fmt.Printf("%s -> %s\n", loc.IP, loc.IPLocate.Location)
}
for _, asn := range info.IPv4AsnInfos {
    fmt.Printf("%s -> AS%d %s\n", asn.IP, asn.OrganisationNumber, asn.OrganisationName)
}
```

### 2. pkg/iplocate - IP 地理位置查询

底层接口，支持直接调用各数据库实现。

```go
import "github.com/winezer0/ipinfo/pkg/iplocate/qqwry"

// 纯真数据库
db, err := qqwry.NewQQWryDB("assets/qqwry.dat")
if err != nil {
    panic(err)
}
defer db.Close()

result := db.FindFull("8.8.8.8")
fmt.Printf("国家: %s, 省份: %s, 城市: %s, ISP: %s\n",
    result.Country, result.Province, result.City, result.ISP)
```

### 3. pkg/asninfo - ASN 信息查询

```go
import "github.com/winezer0/ipinfo/pkg/asninfo/asnmmdb"

config := &asnmmdb.MMDBConfig{
    AsnIpvxDb:            "assets/geolite2-asn.mmdb",
    MaxConcurrentQueries: 100,
}

manager, err := asnmmdb.NewMMDBManager(config)
if err != nil {
    panic(err)
}
defer manager.Close()

if err := manager.Init(); err != nil {
    panic(err)
}

result := manager.FindASN("8.8.8.8")
fmt.Printf("AS%d %s\n", result.OrganisationNumber, result.OrganisationName)

// ASN 反查 IP 段
ipRanges, _ := manager.ASNToIPRanges(15169)
for _, ipNet := range ipRanges {
    fmt.Println(ipNet.String())
}
```

***

## cmd/ipinfoapi - HTTP API 服务

基于标准库 `net/http` 构建的 RESTful API 服务，提供 IP 地理位置和 ASN 信息查询接口。

### API 端点

| 方法  | 路径                      | 说明               | 认证 |
| --- | ----------------------- | ---------------- | -- |
| GET | `/ping`                 | 健康检查             | 不需要 |
| GET | `/ip?q=<ip>`            | 查询 IP 地理位置       | 需要 |
| GET | `/asn?q=<ip>`           | 查询 ASN 信息        | 需要 |
| GET | `/all?q=<ip>`           | 查询完整信息（IP + ASN） | 需要 |
| GET | `/full?q=<ip1,ip2,...>` | 批量查询             | 需要 |

### 认证方式

支持两种认证方式（优先级从高到低）：

1. **Authorization 请求头**：`Authorization: Bearer <token>`
2. **URL 参数**：`?token=<token>`

### 批量限制与可信代理

`http.max_batch_size` 默认为 100，限制 `/full` 每次提交的 IP 数量（去重前）。
`http.trusted_proxies` 默认为空，此时客户端 IP 使用连接地址。部署在反向代理后时，填写代理的 CIDR，例如 `["127.0.0.1/32", "::1/128"]`；只有来自可信代理的转发头才会参与客户端 IP 判断。

### 请求示例

```bash
# 健康检查
curl "http://localhost:8080/ping"

# 查询客户端 IP 完整信息
curl "http://localhost:8080/all?token=your-token"

# 查询 IP 地理位置
curl "http://localhost:8080/ip?q=8.8.8.8&token=your-token"

# 查询 ASN 信息
curl "http://localhost:8080/asn?q=8.8.8.8&token=your-token"

# 查询完整信息
curl "http://localhost:8080/all?q=8.8.8.8&token=your-token"

# 批量查询
curl "http://localhost:8080/full?q=8.8.8.8,1.1.1.1,114.114.114.114&token=your-token"
```

### 响应格式

**成功响应：**

```json
{
  "success": true,
  "code": 0,
  "data": { ... }
}
```

**错误响应：**

```json
{
  "success": false,
  "code": 1001,
  "error": "错误描述"
}
```

### 测试与性能基线

`go test ./...` 默认运行不依赖外部数据库的测试。真实数据库测试统一通过 `internal/testassets` 显式启用，启用后文件缺失或初始化失败会使测试失败。

```powershell
# 下载 17 个集成测试数据库并校验 SHA-256；已有且校验通过的文件直接复用
python scripts/fetch_test_assets.py

# PowerShell：运行真实数据库集成测试和单元测试
$env:IPINFO_INTEGRATION = "1"
go test ./... -count=1
Remove-Item Env:IPINFO_INTEGRATION

# 压测：单次/批量、不同并发度、单 Searcher 与候选池
go run ./cmd/ipbench -xdb assets/ip2region_v4.xdb -requests 10000 -out benchmarks/ipv4-run1.json
```

Linux/macOS 使用 `IPINFO_INTEGRATION=1 go test ./... -count=1`。可通过 `IPINFO_ASSETS_DIR` 指定数据库目录的绝对路径；测试过程不会自动联网下载。

下载清单位于 `scripts/test-assets.json`：仓库文件固定提交版本，IPDB 固定 npm 版本，Release 文件固定 SHA-256。Release 被上游替换时会校验失败，需要人工更新清单；不会静默使用新数据。下载的数据库保留在已忽略的 `assets/`，数据来源和使用条款见各上游项目。

XDB 微基准输出吞吐、P50/P95、分配及近似堆内存；`cmd/apibench` 使用真实 HTTP API、XDB 和 ASN 数据库，另行测量服务进程 CPU、RSS 和错误率。运行及汇总方法见 `benchmarks/PRODUCTION.txt`。

### XDB 查询池与持续集成

API 和两个命令行工具支持顶层配置 `xdb_pool_size: 1`，范围为 1–64，省略或配置为 0 时使用 1。仅作用于 XDB，每个数据库实例共享一份 VectorIndex，并打开相应数量的独立文件句柄。Go 调用方可设置 `IPDbConfig.XDBPoolSize`。默认保持 1；并发查询受 XDB 串行读取限制时，可压测比较 4，再决定是否调整。

GitHub CI 在 push、pull request 和手动触发时运行 Windows/Linux 单元测试、编译、vet 和 Python 测试，Linux 额外运行 race 与短时 fuzz。真实数据库集成测试由每周计划或手动触发，校验固定 SHA-256 并缓存数据库。上游删除或替换固定资源时，需人工更新清单。发布工作流根据 `go.mod` 选择 Go 版本。

### 部署方式

#### 1. 准备数据库文件

将所需的数据库文件放入 `assets/` 目录：

默认数据库文件可从 [cdninfo/assets](https://github.com/winezer0/cdninfo/tree/main/assets) 下载。

```
assets/
├── qqwry.dat              # 纯真 IP 数据库
├── zxipv6wry.db           # 紫薇 IPv6 数据库
├── geolite2-asn.mmdb      # GeoLite2 ASN 数据库
└── ...
```

#### 2. 配置文件

使用 `./ipinfoapi --gen -c ipinfoapi.yaml` 生成完整配置，或复制 `cmd/ipinfoapi/ipinfoapi.yaml`。下面示例使用已准备好的本地数据库：

```yaml
auth:
  token: "your-secret-token"
  enable: true

http:
  enable: true
  port: 8080
  https: false
  read_timeout: 10
  write_timeout: 10
  idle_timeout: 30

databases:
  - module: asn_ipvx_db
    filename: geolite2-asn.mmdb
    storage-dir: assets
    enable: false
  - module: ipv4_locate_db
    filename: qqwry.dat
    storage-dir: assets
    enable: false
  - module: ipv6_locate_db
    filename: zxipv6wry.db
    storage-dir: assets
    enable: false

log:
  level: "info"
  file: "logs/ipinfoapi.log"
  console: "TLCM"
  max_size: 100
  max_backups: 3
```

`databases` 由 `github.com/winezer0/downutils v0.1.1` 处理。`enable: false` 仅关闭下载，查询仍使用配置的文件路径；移除配置项才会取消该模块。启用下载时设置 `enable: true` 和 `download-urls`；`keep-updated` 使用字符串，如 `"false"`（已有文件不更新）、`"7d"`。可选的 `checksum`、`check-size`、`max-size` 分别用于校验和、文件大小校验和下载字节上限。相对路径基于进程工作目录；未设置 `storage-dir` 时文件保存在工作目录。

日志直接使用 `github.com/winezer0/slogs v0.1.2`，保留应用的 `log` 配置字段：`level` 同时控制控制台与文件级别，`file` 为空时不写文件，`console` 支持 `text`、`json`、`off` 或 `TLCM` 等格式掩码。文件日志默认输出 JSON，保留 30 天并压缩备份；大小和备份数由 `max_size`、`max_backups` 控制。下载库与 API 共用该日志实例。

#### 3. 编译

```bash
# Windows
.\build-win.bat

# Linux
./build-linux.bat

# 手动编译
go build -o ipinfoapi ./cmd/ipinfoapi
```

#### 4. 运行

```bash
# 前台运行
./ipinfoapi

# 后台运行（Linux）
nohup ./ipinfoapi &

# 后台运行（Windows）
Start-Process -NoNewWindow .\ipinfoapi.exe
```

## 命令行查询

`iplocate` 查询定位与 ASN，`ipmultiloc` 并行查询多个定位库。分别编辑 `cmd/iplocate/iplocate.yaml` 和 `cmd/ipmultiloc/ipmultiloc.yaml`，仅保留本地已有的数据库路径；相对路径基于当前工作目录。

```bash
go run ./cmd/iplocate -c cmd/iplocate/iplocate.yaml -I 8.8.8.8
go run ./cmd/ipmultiloc -c cmd/ipmultiloc/ipmultiloc.yaml -I 8.8.8.8 -I 2001:4860:4860::8888
```

两个工具均支持 `xdb_pool_size`。`ipmultiloc` 使用 `ip_locate_dbs` 列表配置数据库，不查询 ASN；Go 调用方使用 `pkg/multiloc`。
