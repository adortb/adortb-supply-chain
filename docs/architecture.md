# adortb-supply-chain 内部架构

## 内部架构图

```
┌──────────────────────────────────────────────────────────────────┐
│                   adortb-supply-chain 内部架构                    │
│                                                                  │
│  外部请求                                                        │
│  ├── DSP:       GET /sellers.json                                │
│  ├── 媒体方:    GET /v1/adstxt/template                          │
│  ├── ADX:       POST /v1/schain/build                           │
│  └── 运营:      GET /v1/spo/analyze                             │
│      │                                                          │
│      ▼                                                          │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  internal/api/server.go  NewRouter()                     │   │
│  │  mux 路由表                                               │   │
│  └──────┬────────────┬──────────────┬──────────────┬────────┘   │
│         │            │              │              │            │
│   sellers.json   adstxt         schain           spo           │
│         │            │              │              │            │
│         ▼            ▼              ▼              ▼            │
│  ┌───────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │ sellers   │  │ adstxt/  │  │ schain/  │  │ spo/     │      │
│  │ json/     │  │ validator│  │ builder  │  │ analyzer │      │
│  │ handler   │  │ fetcher  │  │ signer   │  │ dedup    │      │
│  │ generator │  │ parser   │  │ validator│  │          │      │
│  └─────┬─────┘  └────┬─────┘  └──────────┘  └────┬─────┘      │
│        │             │                            │            │
│        └─────────────┴────────────────────────────┘            │
│                             │                                   │
│                    ┌────────▼─────────┐                         │
│                    │  repo/pg_repo.go │                         │
│                    │  PostgreSQL      │                         │
│                    │                  │                         │
│                    │  sellers         │                         │
│                    │  ads_txt_records │                         │
│                    │  publishers      │                         │
│                    └──────────────────┘                         │
│                                                                  │
│  Background Goroutine: AdsTxt Crawler（定时爬取）                │
└──────────────────────────────────────────────────────────────────┘
```

## 数据流

### ads.txt 校验数据流

```
POST /v1/adstxt/validate {domain, account_id}
    │
    ▼
deps.Fetcher.FetchDomain(ctx, domain)        实时爬取
    GET https://{domain}/ads.txt
    → adstxt.Parse(content) → ParseResult
    → validator.ingest(records) → 内存更新
    │
    ▼
deps.Validator.CheckAnyDeclared(domain)
    → 检查内存中是否存在 Record{Domain="adortb.com", ...}
    → bool (declared)
    │
    ▼
ValidationResult{Valid, Declared, Issues}
```

### SupplyChain 构造数据流

```
POST /v1/schain/build {publisher_asi, publisher_sid, nodes, add_self}
    │
    ▼
SchainBuilder.Build(BuildRequest, addSelf)
    nodes = []
    nodes += Node{ASI: publisherASI, SID: publisherSID, HP: 1}  // publisher 节点
    nodes += req.ExtraNodes                                        // 中间 reseller
    if addSelf:
        nodes += Node{ASI: adxASI, SID: adxSID, HP: 1}           // 本 ADX 节点
    → SupplyChain{Complete:1, Nodes:nodes, Ver:"1.0"}
    │
    ▼
SchainSigner.Sign(sc)
    → HMAC/签名（防止下游篡改）
    → SignedSupplyChain
    │
    ▼
返回 JSON（注入 OpenRTB bid request source.ext.schain）
```

### sellers.json 生成数据流

```
GET /sellers.json
    │
    ▼
sellersjson.Handler.ServeHTTP()
    │
    ▼
Generator.Generate(ctx)
    → pg.ListSellers() → []SellerInfo{SellerID, Name, Domain, Type}
    → 组装 sellers.json 规范格式
    → JSON 响应（ETag 缓存）
```

### ads.txt 定时爬虫

```
startCrawler(intervalHours=6)
    │ 启动时立即执行一次
    ▼
crawlOnce()
    → pg.ListUndeclaredDomains(ctx) → []string{domain}
    → fetcher.FetchAll(ctx, domains)
        for each domain（并发）:
            GET https://{domain}/ads.txt
            Parse() → Records
            validator.ingest(domain, records)
            pg.UpsertAdsTxtRecords(domain, records)
```

## 时序图

```
DSP         Router   SellersHandler  Generator  PostgreSQL
 │             │           │              │          │
 │─GET sellers.json──────► │              │          │
 │             │           │──Generate()─►│          │
 │             │           │              │─ListSellers►
 │             │           │              │◄─sellers──│
 │             │           │◄─sellers.json│           │
 │◄─JSON───────│           │              │          │

adortb-adx  Router   SchainBuilder  SchainSigner
    │           │           │              │
    │─POST /build──────────►│              │
    │           │           │  Build()     │
    │           │           │  + ExtraNodes│
    │           │──Sign()────────────────► │
    │           │◄─SignedSC────────────── │
    │◄─schain───│           │              │
```

## 状态机

### ads.txt 验证流程

```
域名注册 → 未校验
    │
    ▼
FetchDomain() → 爬取 ads.txt
    │
    ├── 爬取成功 → Parse() → 有本 ADX 记录？
    │               ├── YES → declared=true
    │               └── NO  → declared=false（警告媒体方补充）
    │
    └── 爬取失败（超时/404）→ 记录失败，下次重试
```

### SupplyChain 节点类型

```
SupplyChain.Nodes:
    [0] Publisher Node (HP=1)  总是第一个
    [1..N-2] Extra Nodes       reseller 中间节点（可选）
    [N-1] ADX Node (HP=1)     add_self=true 时追加

HP (Header Payment) = 1: 该节点参与最终交易决策
```

### sellers.json Seller 类型

```
seller_type:
    "PUBLISHER"  → 直接媒体方
    "INTERMEDIARY" → 中间代理/SSP
    "BOTH"       → 兼具两种角色
```
