package adstxt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/adortb/adortb-supply-chain/internal/repo"
)

const (
	fetchTimeout  = 10 * time.Second
	maxBodyBytes  = 512 * 1024 // 512KB
	crawlStatusOK = "ok"
)

// Fetcher 定期抓取 publisher 域名的 /ads.txt 并持久化校验结果。
type Fetcher struct {
	repo      repo.Repository
	validator *Validator
	client    *http.Client
	logger    *slog.Logger
	adxDomain string
}

// NewFetcher 创建 Fetcher。
func NewFetcher(r repo.Repository, v *Validator, adxDomain string, logger *slog.Logger) *Fetcher {
	return &Fetcher{
		repo:      r,
		validator: v,
		adxDomain: adxDomain,
		client: &http.Client{
			Timeout: fetchTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 3 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		logger: logger,
	}
}

// FetchDomain 抓取单个域名的 /ads.txt 并更新数据库。
func (f *Fetcher) FetchDomain(ctx context.Context, domain string) error {
	rawURL := buildAdsTxtURL(domain)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return f.saveError(ctx, domain, "network_error", err)
	}
	req.Header.Set("User-Agent", "AdortbSupplyChainCrawler/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return f.saveError(ctx, domain, "network_error", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return f.saveError(ctx, domain, "not_found", fmt.Errorf("HTTP 404"))
	}
	if resp.StatusCode != http.StatusOK {
		return f.saveError(ctx, domain, "network_error",
			fmt.Errorf("unexpected status: %d", resp.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return f.saveError(ctx, domain, "network_error", err)
	}

	content := string(body)
	parsed := Parse(content)
	declared := f.validator.CheckAnyDeclared(content)

	if parsed.HasError {
		return f.saveRecord(ctx, domain, "parse_error", &declared, content, parsed.Records)
	}
	return f.saveRecord(ctx, domain, crawlStatusOK, &declared, content, parsed.Records)
}

// FetchAll 批量抓取所有已知 publisher 域名。
func (f *Fetcher) FetchAll(ctx context.Context, domains []string) {
	for _, d := range domains {
		if err := f.FetchDomain(ctx, d); err != nil {
			f.logger.Warn("fetch ads.txt failed", "domain", d, "error", err)
		}
	}
}

func (f *Fetcher) saveError(ctx context.Context, domain, status string, err error) error {
	now := time.Now()
	declaredFalse := false
	upsertErr := f.repo.UpsertAdsTxtRecord(ctx, &repo.AdsTxtRecord{
		PublisherDomain: domain,
		LastCrawlAt:     &now,
		LastCrawlStatus: status,
		ADXDeclared:     &declaredFalse,
	})
	if upsertErr != nil {
		f.logger.Error("upsert adstxt error record", "domain", domain, "err", upsertErr)
	}
	return fmt.Errorf("fetch %s: %w", domain, err)
}

func (f *Fetcher) saveRecord(ctx context.Context, domain, status string,
	declared *bool, raw string, records []Record) error {

	now := time.Now()
	recsJSON, _ := json.Marshal(records)
	return f.repo.UpsertAdsTxtRecord(ctx, &repo.AdsTxtRecord{
		PublisherDomain: domain,
		LastCrawlAt:     &now,
		LastCrawlStatus: status,
		ADXDeclared:     declared,
		RawContent:      raw,
		Records:         json.RawMessage(recsJSON),
	})
}

func buildAdsTxtURL(domain string) string {
	if !strings.HasPrefix(domain, "http") {
		return "https://" + domain + "/ads.txt"
	}
	return domain + "/ads.txt"
}
