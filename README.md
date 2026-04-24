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
│   ├── iputils/             # IP 工具函数
│   ├── logging/             # 日志模块
│   └── utils/               # 通用工具
├── cmd/                     # 可执行程序
│   ├── iplocate/            # 命令行 IP 查询工具
│   ├── iplocate2/           # 命令行工具（多数据库版本）
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

### 部署方式

#### 1. 准备数据库文件

将所需的数据库文件放入 `assets/` 目录：

```
assets/
├── qqwry.dat              # 纯真 IP 数据库
├── zxipv6wry.db           # 紫薇 IPv6 数据库
├── geolite2-asn.mmdb      # GeoLite2 ASN 数据库
└── ...
```

#### 2. 配置文件

复制并编辑配置文件 `ipinfoapi.yaml`：

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

database:
  asn_ipv_x_db: "assets/geolite2-asn.mmdb"
  ipv4_locate_db: "assets/qqwry.dat"
  ipv6_locate_db: "assets/zxipv6wry.db"

log:
  level: "info"
  file: "logs/ipinfoapi.log"
  console: "TLCM"
  max_size: 100
  max_backups: 3
```

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
