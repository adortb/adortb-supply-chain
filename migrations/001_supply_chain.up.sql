-- 供应链透明度核心表

CREATE TABLE IF NOT EXISTS sellers (
    seller_id        VARCHAR(64) PRIMARY KEY,
    publisher_id     BIGINT,
    name             VARCHAR(255) NOT NULL,
    domain           VARCHAR(255),
    seller_type      VARCHAR(20)  DEFAULT 'PUBLISHER',
    is_confidential  BOOLEAN      DEFAULT FALSE,
    comment          TEXT,
    status           VARCHAR(20)  DEFAULT 'active',
    created_at       TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sellers_publisher_id ON sellers(publisher_id);
CREATE INDEX IF NOT EXISTS idx_sellers_status ON sellers(status);

CREATE TABLE IF NOT EXISTS adstxt_records (
    id                BIGSERIAL PRIMARY KEY,
    publisher_domain  VARCHAR(255) NOT NULL,
    last_crawl_at     TIMESTAMPTZ,
    last_crawl_status VARCHAR(20),
    adx_declared      BOOLEAN,
    raw_content       TEXT,
    records           JSONB,
    UNIQUE (publisher_domain)
);

CREATE INDEX IF NOT EXISTS idx_adstxt_adx_declared ON adstxt_records(adx_declared);

CREATE TABLE IF NOT EXISTS supply_paths (
    id            BIGSERIAL PRIMARY KEY,
    request_hash  VARCHAR(64),
    path          JSONB,
    depth         INT,
    quality_score DECIMAL(3,2),
    seen_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supply_paths_request_hash ON supply_paths(request_hash);
CREATE INDEX IF NOT EXISTS idx_supply_paths_seen_at ON supply_paths(seen_at);
