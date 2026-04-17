// Package sellersjson 实现 IAB sellers.json 标准的生成与序列化。
package sellersjson

import (
	"context"
	"fmt"
	"time"

	"github.com/adortb/adortb-supply-chain/internal/repo"
)

// SellerType 定义 sellers.json 中的 seller 类型。
type SellerType string

const (
	SellerTypePublisher    SellerType = "PUBLISHER"
	SellerTypeIntermediary SellerType = "INTERMEDIARY"
	SellerTypeBoth         SellerType = "BOTH"
)

// SellerEntry 对应 sellers.json 中一条 seller 记录（IAB sellers.json 1.0）。
type SellerEntry struct {
	SellerID       string     `json:"seller_id"`
	Name           string     `json:"name,omitempty"`
	Domain         string     `json:"domain,omitempty"`
	SellerType     SellerType `json:"seller_type"`
	IsConfidential int        `json:"is_confidential"` // 0 or 1
	Comment        string     `json:"comment,omitempty"`
}

// SellersJSON 是完整的 sellers.json 响应体。
type SellersJSON struct {
	ContactEmail    string        `json:"contact_email"`
	ContactAddress  string        `json:"contact_address,omitempty"`
	Version         string        `json:"version"`
	Identifiers     []interface{} `json:"identifiers"`
	Sellers         []SellerEntry `json:"sellers"`
	GeneratedAt     time.Time     `json:"ext_generated_at,omitempty"` // 扩展字段，非标准
}

// Generator 从数据库查询并生成 sellers.json 内容。
type Generator struct {
	repo         repo.Repository
	contactEmail string
}

// NewGenerator 创建生成器。
func NewGenerator(r repo.Repository, contactEmail string) *Generator {
	return &Generator{repo: r, contactEmail: contactEmail}
}

// Generate 查询所有 active seller 并构建 SellersJSON 对象。
func (g *Generator) Generate(ctx context.Context) (*SellersJSON, error) {
	sellers, err := g.repo.ListActiveSellers(ctx)
	if err != nil {
		return nil, fmt.Errorf("sellersjson: list sellers: %w", err)
	}

	entries := make([]SellerEntry, 0, len(sellers))
	for _, s := range sellers {
		entry := SellerEntry{
			SellerID:   s.SellerID,
			Name:       s.Name,
			Domain:     s.Domain,
			SellerType: toSellerType(s.SellerType),
		}
		if s.IsConfidential {
			entry.IsConfidential = 1
			// is_confidential=1 时 name/domain 必须隐藏
			entry.Name = ""
			entry.Domain = ""
		}
		if s.Comment != "" {
			entry.Comment = s.Comment
		}
		entries = append(entries, entry)
	}

	return &SellersJSON{
		ContactEmail: g.contactEmail,
		Version:      "1.0",
		Identifiers:  []interface{}{},
		Sellers:      entries,
		GeneratedAt:  time.Now().UTC(),
	}, nil
}

func toSellerType(s string) SellerType {
	switch SellerType(s) {
	case SellerTypePublisher, SellerTypeIntermediary, SellerTypeBoth:
		return SellerType(s)
	default:
		return SellerTypePublisher
	}
}
