// Package client 提供 adortb-supply-chain 服务的 HTTP 客户端，供其他服务调用。
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultTimeout = 5 * time.Second

// SupplyChainClient 是 supply-chain 服务的 HTTP 客户端。
type SupplyChainClient struct {
	baseURL string
	http    *http.Client
}

// New 创建客户端，baseURL 例如 "http://supply-chain:8091"。
func New(baseURL string) *SupplyChainClient {
	return &SupplyChainClient{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Node 对应 schain 节点（与 schain 包保持一致的结构，避免循环依赖）。
type Node struct {
	ASI    string `json:"asi"`
	SID    string `json:"sid"`
	HP     int    `json:"hp"`
	RID    string `json:"rid,omitempty"`
	Name   string `json:"name,omitempty"`
	Domain string `json:"domain,omitempty"`
}

// SupplyChain 是 schain 对象。
type SupplyChain struct {
	Complete  int    `json:"complete"`
	Nodes     []Node `json:"nodes"`
	Ver       string `json:"ver"`
	Signature string `json:"ext_sig,omitempty"`
}

// BuildSchainRequest 是 POST /v1/schain/build 的请求体。
type BuildSchainRequest struct {
	PublisherASI string `json:"publisher_asi"`
	PublisherSID string `json:"publisher_sid"`
	AddSelf      bool   `json:"add_self"`
}

// BuildSchain 调用 supply-chain 服务构建 SupplyChain 对象。
func (c *SupplyChainClient) BuildSchain(ctx context.Context, req BuildSchainRequest) (*SupplyChain, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("client: marshal request: %w", err)
	}

	resp, err := c.post(ctx, "/v1/schain/build", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("client: schain build status %d", resp.StatusCode)
	}

	var sc SupplyChain
	if err := json.NewDecoder(resp.Body).Decode(&sc); err != nil {
		return nil, fmt.Errorf("client: decode schain: %w", err)
	}
	return &sc, nil
}

// AdsTxtStatus 表示 publisher 的 ads.txt 声明状态。
type AdsTxtStatus struct {
	Domain   string `json:"domain"`
	Valid    bool   `json:"valid"`
	Declared bool   `json:"declared"`
	Issues   []string `json:"issues,omitempty"`
}

// ValidateAdsTxt 检查指定域名是否正确声明了本 ADX。
func (c *SupplyChainClient) ValidateAdsTxt(ctx context.Context, domain, accountID string) (*AdsTxtStatus, error) {
	body, err := json.Marshal(map[string]string{
		"domain":     domain,
		"account_id": accountID,
	})
	if err != nil {
		return nil, fmt.Errorf("client: marshal request: %w", err)
	}

	resp, err := c.post(ctx, "/v1/adstxt/validate", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("client: adstxt validate status %d", resp.StatusCode)
	}

	var status AdsTxtStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("client: decode adstxt status: %w", err)
	}
	return &status, nil
}

// Health 调用 /health 检查服务健康状态。
func (c *SupplyChainClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("client: health request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("client: health: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("client: health status %d", resp.StatusCode)
	}
	return nil
}

func (c *SupplyChainClient) post(ctx context.Context, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("client: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client: post %s: %w", path, err)
	}
	return resp, nil
}

// readAndClose 读取并关闭响应体（防止连接泄露）。
func readAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}

// WithTimeout 返回一个使用指定超时的新客户端（覆盖默认值）。
func (c *SupplyChainClient) WithTimeout(d time.Duration) *SupplyChainClient {
	return &SupplyChainClient{
		baseURL: c.baseURL,
		http:    &http.Client{Timeout: d},
	}
}
