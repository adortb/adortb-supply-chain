# adortb-supply-chain

> adortb 平台的广告供应链透明度服务，实现 IAB 标准的 ads.txt 校验、sellers.json 生成、SupplyChain Object 构造（OpenRTB schain）和供应路径优化（SPO）分析，打击欺诈流量，保护广告主利益。

## 架构定位

```
┌─────────────────────────────────────────────────────────────────┐
│                      adortb 平台整体架构                         │
│                                                                  │
│  广告主 / DSP                                                    │
│       │ 验证供应链合规性                                         │
│       ▼                                                         │
│  ★ adortb-supply-chain (Supply Chain Service)                  │
│       │                                                         │
│  ┌────┼──────────────────────────────────────────────┐          │
│  │    ▼                                             │          │
│  │  [AdsTxt Crawler]   周期爬取媒体方 ads.txt         │          │
│  │       ↓                                          │          │
│  │  [Validator]        校验本 ADX 是否被声明授权      │          │
│  │       ↓                                          │          │
│  │  [SChain Builder]   构造 OpenRTB schain 对象      │          │
│  │       ↓                                          │          │
│  │  [sellers.json]     生成本 ADX 的卖方信息文件      │          │
│  │       ↓                                          │          │
│  │  [SPO Analyzer]     分析供应路径，优化路由          │          │
│  └─────────────────────────────────────────────────┘          │
│                                                                  │
│  GET /sellers.json  ←  DSP 验证 ADX 合规性                      │
│  PostgreSQL（sellers/ads.txt 记录）                             │
└─────────────────────────────────────────────────────────────────┘
```

Supply Chain 服务是平台**广告欺诈防护**的基础设施，确保每一笔广告交易都可验证、可追溯。

## 目录结构

```
adortb-supply-chain/
├── go.mod                          # Go 1.25.3，依赖 lib/pq、prometheus
├── cmd/supply-chain/
│   └── main.go                     # 主程序：服务初始化、定时爬虫启动
├── client/                         # Go 客户端
├── migrations/                     # PostgreSQL 迁移 SQL
└── internal/
    ├── api/
    │   └── server.go               # HTTP 路由注册（sellers.json/adstxt/schain/spo）
    ├── adstxt/
    │   ├── parser.go               # ads.txt 解析器（Record/ParseResult，支持 DIRECT/RESELLER）
    │   ├── validator.go            # 校验器（检查本 ADX 是否被媒体方声明）
    │   └── fetcher.go              # HTTP 爬虫（批量爬取 https://domain/ads.txt）
    ├── sellersjson/
    │   ├── generator.go            # sellers.json 生成（IAB sellers.json 规范）
    │   └── handler.go              # GET /sellers.json 处理器
    ├── schain/
    │   ├── builder.go              # SupplyChain Object 构造（DIRECT / reseller 场景）
    │   ├── validator.go            # schain 合法性校验
    │   └── signer.go               # schain 签名（防篡改）
    ├── spo/
    │   ├── analyzer.go             # 供应路径优化分析
    │   └── dedup.go                # 重复路径去重
    ├── config/
    │   └── config.go               # 配置加载（ADX 域名/CertAuth/端口/爬虫间隔）
    ├── repo/
    │   └── pg_repo.go              # PostgreSQL 数据访问
    └── metrics/
        └── metrics.go              # Prometheus 指标
```

## 快速开始

### 环境要求

- Go 1.25.3
- PostgreSQL

```bash
export PATH="$HOME/.goenv/versions/1.25.3/bin:$PATH"
```

### 运行服务

```bash
cd adortb-supply-chain

export DATABASE_URL="postgres://postgres:postgres@localhost:5432/adortb_schain?sslmode=disable"
export ADX_DOMAIN="adortb.com"
export ADX_CERT_AUTH="abc123certauth"
export PORT=8095
export CRAWL_INTERVAL_HOURS=6

go run cmd/supply-chain/main.go
```

### 运行测试

```bash
go test ./... -cover -race
```

## HTTP API

### GET /sellers.json

返回本 ADX 的 IAB sellers.json 文件，供 DSP 验证卖家信息。

```json
{
  "contact_email": "ops@adortb.com",
  "version": "1.0",
  "sellers": [
    {
      "seller_id": "pub-001",
      "name": "Example Publisher",
      "domain": "example.com",
      "seller_type": "PUBLISHER"
    }
  ]
}
```

### GET /v1/adstxt/template?publisher_id={seller_id}

获取媒体方应添加到其 ads.txt 的授权行。

**响应**：

```json
{
  "line": "adortb.com, pub-001, DIRECT, abc123certauth",
  "description": "Add this line to your ads.txt file at the root of your domain"
}
```

### POST /v1/adstxt/validate

实时验证媒体方域名的 ads.txt 是否包含本 ADX 的声明。

```json
{
  "domain": "publisher.com",
  "account_id": "pub-001"
}
```

### POST /v1/schain/build

构造 OpenRTB SupplyChain Object（注入 bid request 的 `source.ext.schain`）。

```json
{
  "publisher_asi": "publisher.com",
  "publisher_sid": "pub-001",
  "add_self": true
}
```

**响应**：

```json
{
  "complete": 1,
  "ver": "1.0",
  "nodes": [
    {"asi": "publisher.com", "sid": "pub-001", "hp": 1},
    {"asi": "adortb.com", "sid": "adortb-adx", "hp": 1}
  ]
}
```

### GET /v1/spo/analyze?publisher_id={id}

分析指定发布商的供应路径，识别重复/冗余路由。

## 配置说明

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `DATABASE_URL` | — | PostgreSQL 连接串 |
| `ADX_DOMAIN` | — | 本 ADX 的域名（用于 ads.txt 声明 + schain） |
| `ADX_CERT_AUTH` | — | TAG 认证 ID |
| `PORT` | — | 监听端口 |
| `CRAWL_INTERVAL_HOURS` | — | ads.txt 爬虫间隔（小时） |
| `SCHAIN_SIGN_KEY` | — | schain 签名密钥 |

## ads.txt 格式说明

```
# 标准格式：domain, account_id, relationship[, cert_auth]
adortb.com, pub-001, DIRECT, abc123
exchange.com, reseller-456, RESELLER
```

- `DIRECT`：直接授权，广告主与媒体方直接交易
- `RESELLER`：转售授权，通过中间方交易

## 相关项目

| 项目 | 说明 |
|------|------|
| [adortb-adx](https://github.com/adortb/adortb-adx) | 竞价引擎，注入 schain 对象 |
| [adortb-brand-safety](https://github.com/adortb/adortb-brand-safety) | 品牌安全 |
| [adortb-infra](https://github.com/adortb/adortb-infra) | 基础设施 |
