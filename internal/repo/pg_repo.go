// Package repo 提供 PostgreSQL 数据访问层。
package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Seller 对应 sellers 表记录。
type Seller struct {
	SellerID       string
	PublisherID    int64
	Name           string
	Domain         string
	SellerType     string
	IsConfidential bool
	Comment        string
	Status         string
	CreatedAt      time.Time
}

// AdsTxtRecord 对应 adstxt_records 表记录。
type AdsTxtRecord struct {
	ID              int64
	PublisherDomain string
	LastCrawlAt     *time.Time
	LastCrawlStatus string
	ADXDeclared     *bool
	RawContent      string
	Records         json.RawMessage
}

// SupplyPath 对应 supply_paths 表记录。
type SupplyPath struct {
	ID           int64
	RequestHash  string
	Path         json.RawMessage
	Depth        int
	QualityScore float64
	SeenAt       time.Time
}

// Repository 定义数据访问接口，便于测试替换。
type Repository interface {
	ListActiveSellers(ctx context.Context) ([]*Seller, error)
	GetSellerByID(ctx context.Context, sellerID string) (*Seller, error)
	UpsertAdsTxtRecord(ctx context.Context, rec *AdsTxtRecord) error
	GetAdsTxtRecord(ctx context.Context, domain string) (*AdsTxtRecord, error)
	ListUndeclaredDomains(ctx context.Context) ([]string, error)
	InsertSupplyPath(ctx context.Context, sp *SupplyPath) error
	ListSupplyPaths(ctx context.Context, requestHash string) ([]*SupplyPath, error)
}

// PGRepo 是 Repository 的 PostgreSQL 实现。
type PGRepo struct {
	db *sql.DB
}

// New 创建 PGRepo，验证连通性。
func New(databaseURL string) (*PGRepo, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("repo: open db: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("repo: ping db: %w", err)
	}
	return &PGRepo{db: db}, nil
}

// Close 关闭数据库连接池。
func (r *PGRepo) Close() error { return r.db.Close() }

// ListActiveSellers 查询所有 active 状态的 seller。
func (r *PGRepo) ListActiveSellers(ctx context.Context) ([]*Seller, error) {
	const q = `SELECT seller_id, publisher_id, name, COALESCE(domain,''), seller_type,
		is_confidential, COALESCE(comment,''), status, created_at
		FROM sellers WHERE status = 'active' ORDER BY seller_id`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("repo: list sellers: %w", err)
	}
	defer rows.Close()

	var sellers []*Seller
	for rows.Next() {
		s := &Seller{}
		if err := rows.Scan(&s.SellerID, &s.PublisherID, &s.Name, &s.Domain,
			&s.SellerType, &s.IsConfidential, &s.Comment, &s.Status, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("repo: scan seller: %w", err)
		}
		sellers = append(sellers, s)
	}
	return sellers, rows.Err()
}

// GetSellerByID 按 seller_id 查询。
func (r *PGRepo) GetSellerByID(ctx context.Context, sellerID string) (*Seller, error) {
	const q = `SELECT seller_id, publisher_id, name, COALESCE(domain,''), seller_type,
		is_confidential, COALESCE(comment,''), status, created_at
		FROM sellers WHERE seller_id = $1`

	s := &Seller{}
	err := r.db.QueryRowContext(ctx, q, sellerID).Scan(
		&s.SellerID, &s.PublisherID, &s.Name, &s.Domain,
		&s.SellerType, &s.IsConfidential, &s.Comment, &s.Status, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("repo: get seller: %w", err)
	}
	return s, nil
}

// UpsertAdsTxtRecord 插入或更新 ads.txt 爬取记录。
func (r *PGRepo) UpsertAdsTxtRecord(ctx context.Context, rec *AdsTxtRecord) error {
	const q = `INSERT INTO adstxt_records
		(publisher_domain, last_crawl_at, last_crawl_status, adx_declared, raw_content, records)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (publisher_domain) DO UPDATE SET
		last_crawl_at = EXCLUDED.last_crawl_at,
		last_crawl_status = EXCLUDED.last_crawl_status,
		adx_declared = EXCLUDED.adx_declared,
		raw_content = EXCLUDED.raw_content,
		records = EXCLUDED.records`

	_, err := r.db.ExecContext(ctx, q,
		rec.PublisherDomain, rec.LastCrawlAt, rec.LastCrawlStatus,
		rec.ADXDeclared, rec.RawContent, rec.Records)
	if err != nil {
		return fmt.Errorf("repo: upsert adstxt: %w", err)
	}
	return nil
}

// GetAdsTxtRecord 按域名查询 ads.txt 记录。
func (r *PGRepo) GetAdsTxtRecord(ctx context.Context, domain string) (*AdsTxtRecord, error) {
	const q = `SELECT id, publisher_domain, last_crawl_at, last_crawl_status,
		adx_declared, COALESCE(raw_content,''), COALESCE(records,'null'::jsonb)
		FROM adstxt_records WHERE publisher_domain = $1`

	rec := &AdsTxtRecord{}
	err := r.db.QueryRowContext(ctx, q, domain).Scan(
		&rec.ID, &rec.PublisherDomain, &rec.LastCrawlAt, &rec.LastCrawlStatus,
		&rec.ADXDeclared, &rec.RawContent, &rec.Records)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("repo: get adstxt: %w", err)
	}
	return rec, nil
}

// ListUndeclaredDomains 返回未声明本 ADX 的 publisher 域名列表。
func (r *PGRepo) ListUndeclaredDomains(ctx context.Context) ([]string, error) {
	const q = `SELECT publisher_domain FROM adstxt_records
		WHERE adx_declared = FALSE OR adx_declared IS NULL`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("repo: list undeclared: %w", err)
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, fmt.Errorf("repo: scan domain: %w", err)
		}
		domains = append(domains, d)
	}
	return domains, rows.Err()
}

// InsertSupplyPath 插入一条供应路径记录。
func (r *PGRepo) InsertSupplyPath(ctx context.Context, sp *SupplyPath) error {
	const q = `INSERT INTO supply_paths (request_hash, path, depth, quality_score, seen_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.ExecContext(ctx, q,
		sp.RequestHash, sp.Path, sp.Depth, sp.QualityScore, sp.SeenAt)
	if err != nil {
		return fmt.Errorf("repo: insert supply_path: %w", err)
	}
	return nil
}

// ListSupplyPaths 按 request_hash 查询供应路径。
func (r *PGRepo) ListSupplyPaths(ctx context.Context, requestHash string) ([]*SupplyPath, error) {
	const q = `SELECT id, request_hash, path, depth, quality_score, seen_at
		FROM supply_paths WHERE request_hash = $1 ORDER BY seen_at DESC LIMIT 100`

	rows, err := r.db.QueryContext(ctx, q, requestHash)
	if err != nil {
		return nil, fmt.Errorf("repo: list supply_paths: %w", err)
	}
	defer rows.Close()

	var paths []*SupplyPath
	for rows.Next() {
		sp := &SupplyPath{}
		if err := rows.Scan(&sp.ID, &sp.RequestHash, &sp.Path,
			&sp.Depth, &sp.QualityScore, &sp.SeenAt); err != nil {
			return nil, fmt.Errorf("repo: scan supply_path: %w", err)
		}
		paths = append(paths, sp)
	}
	return paths, rows.Err()
}
