# adortb-supply-chain

> adortb 平台供应链透明度服务，实现 IAB ads.txt 爬取校验、sellers.json 生成、OpenRTB SupplyChain Object 构造和供应路径优化（SPO）分析。

## 快速理解

- **本项目做什么**：爬取媒体方 ads.txt 验证授权声明，生成本 ADX 的 sellers.json，在竞拍时为 bid request 注入 schain 对象，分析供应路径冗余
- **架构位置**：平台反欺诈/合规基础设施，为 adortb-adx 竞拍链路提供供应链验证
- **核心入口**：
  - 服务启动：`cmd/supply-chain/main.go`
  - HTTP 路由：`internal/api/server.go:NewRouter`
  - SupplyChain 构造：`internal/schain/builder.go:Builder.Build`

## 目录结构

```
adortb-supply-chain/
├── cmd/supply-chain/main.go    # 主程序：组件初始化，定时爬虫启动
└── internal/
    ├── api/server.go           # HTTP 路由（sellers.json/adstxt/schain/spo/metrics）
    ├── adstxt/
    │   ├── parser.go           # ads.txt 解析（Parse → ParseResult，DIRECT/RESELLER）
    │   ├── validator.go        # 校验本 ADX 是否被声明授权（Validator.CheckAnyDeclared）
    │   └── fetcher.go          # HTTP 爬虫（FetchDomain/FetchAll，批量爬取）
    ├── sellersjson/
    │   ├── generator.go        # sellers.json 生成（IAB SellerInfo 规范）
    │   └── handler.go          # GET /sellers.json 处理器（ETag 缓存）
    ├── schain/
    │   ├── builder.go          # Builder.Build：publisher node + extra nodes + ADX node
    │   ├── validator.go        # SupplyChain 合法性校验
    │   └── signer.go           # Signer.Sign：防篡改签名
    ├── spo/
    │   ├── analyzer.go         # SPO 分析（重复路径检测，优化建议）
    │   └── dedup.go            # 路径去重
    ├── config/config.go        # 配置加载（ADX_DOMAIN/ADX_CERT_AUTH/PORT/...）
    └── repo/pg_repo.go         # PostgreSQL（sellers/ads.txt 记录）
```

## 核心概念

### ads.txt 解析（`adstxt/parser.go:Parse`）

```go
// 标准格式：domain, account_id, DIRECT[, cert_auth]
// 返回 ParseResult（Records + Errors，行级错误不中断解析）
result := adstxt.Parse(content)
for _, rec := range result.Records {
    // rec.Domain, rec.AccountID, rec.Rel (DIRECT/RESELLER), rec.CertAuthID
}
```

### SupplyChain Object 构造（`schain/builder.go:Builder.Build`）

```go
// DIRECT 场景（单节点）
sc := builder.BuildDirect(publisherASI, publisherSID)

// Reseller 场景（多节点）
sc := builder.Build(BuildRequest{
    PublisherASI: "publisher.com",
    PublisherSID: "pub-001",
    ExtraNodes:   []Node{{ASI: "ssp.com", SID: "reseller-123", HP: 1}},
}, addSelf=true)
// 结果：publisher → ssp → adortb-adx（complete=1）
```

### 定时爬虫

`cmd/supply-chain/main.go:startCrawler` 启动时立即执行一次，之后按 `CRAWL_INTERVAL_HOURS` 周期爬取。每次调用 `pg.ListUndeclaredDomains` 获取未校验的域名列表，批量并发爬取 `https://{domain}/ads.txt`。

## 开发指南

### Go 版本

```bash
export PATH="$HOME/.goenv/versions/1.25.3/bin:$PATH"
```

### 本地运行

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/adortb_schain?sslmode=disable"
export ADX_DOMAIN="adortb.com"
export ADX_CERT_AUTH="abc123"
export PORT=8095
go run cmd/supply-chain/main.go

# 获取 ads.txt 模板行
curl "http://localhost:8095/v1/adstxt/template?publisher_id=pub-001"

# 构造 schain
curl -X POST http://localhost:8095/v1/schain/build \
  -d '{"publisher_asi":"pub.com","publisher_sid":"pub-001","add_self":true}'

# 获取 sellers.json
curl http://localhost:8095/sellers.json
```

### 测试

```bash
go test ./... -cover -race
go test ./internal/adstxt/... -v    # parser/validator 测试
go test ./internal/schain/... -v    # builder/signer 测试
go test ./internal/spo/... -v       # SPO 分析测试
```

### 代码约定

- `ads.txt` 解析是**容错的**：行级格式错误记录到 `ParseResult.Errors` 但不中断
- `Builder` 是无状态的（adxASI/adxSID 在构造时注入），可复用
- `sellers.json` 端点应支持高频访问（DSP 定期拉取），实现了 ETag 缓存

## 依赖关系

- **上游**：DSP（拉取 sellers.json），adortb-adx（注入 schain）
- **下游**：PostgreSQL，媒体方服务器（爬取 ads.txt）
- **依赖的库**：`lib/pq`，`prometheus`

## 深入阅读

- ads.txt 变量行过滤（CONTACT/SUBDOMAIN）：`adstxt/parser.go:isVariableLine`
- sellers.json 规范结构：`sellersjson/generator.go`
- SPO 重复路径分析算法：`spo/analyzer.go:Analyzer.Analyze`
- 配置加载（含环境变量列表）：`config/config.go`
