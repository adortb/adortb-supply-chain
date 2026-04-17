package sellersjson

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/adortb/adortb-supply-chain/internal/repo"
)

// mockRepo 是 repo.Repository 的内存 mock 实现。
type mockRepo struct {
	sellers []*repo.Seller
}

func (m *mockRepo) ListActiveSellers(_ context.Context) ([]*repo.Seller, error) {
	return m.sellers, nil
}
func (m *mockRepo) GetSellerByID(_ context.Context, _ string) (*repo.Seller, error) {
	return nil, nil
}
func (m *mockRepo) UpsertAdsTxtRecord(_ context.Context, _ *repo.AdsTxtRecord) error { return nil }
func (m *mockRepo) GetAdsTxtRecord(_ context.Context, _ string) (*repo.AdsTxtRecord, error) {
	return nil, nil
}
func (m *mockRepo) ListUndeclaredDomains(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockRepo) InsertSupplyPath(_ context.Context, _ *repo.SupplyPath) error { return nil }
func (m *mockRepo) ListSupplyPaths(_ context.Context, _ string) ([]*repo.SupplyPath, error) {
	return nil, nil
}

func TestGenerate_Structure(t *testing.T) {
	mock := &mockRepo{
		sellers: []*repo.Seller{
			{
				SellerID:   "pub-123",
				Name:       "News Inc",
				Domain:     "news.com",
				SellerType: "PUBLISHER",
				Status:     "active",
				CreatedAt:  time.Now(),
			},
		},
	}

	gen := NewGenerator(mock, "ops@adortb.com")
	result, err := gen.Generate(context.Background())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if result.Version != "1.0" {
		t.Errorf("version: got %q", result.Version)
	}
	if result.ContactEmail != "ops@adortb.com" {
		t.Errorf("contact_email: got %q", result.ContactEmail)
	}
	if len(result.Sellers) != 1 {
		t.Fatalf("expected 1 seller, got %d", len(result.Sellers))
	}

	s := result.Sellers[0]
	if s.SellerID != "pub-123" {
		t.Errorf("seller_id: got %q", s.SellerID)
	}
	if s.SellerType != SellerTypePublisher {
		t.Errorf("seller_type: got %q", s.SellerType)
	}
}

func TestGenerate_ConfidentialHidesInfo(t *testing.T) {
	mock := &mockRepo{
		sellers: []*repo.Seller{
			{
				SellerID:       "priv-001",
				Name:           "Secret Publisher",
				Domain:         "secret.com",
				SellerType:     "PUBLISHER",
				IsConfidential: true,
				Status:         "active",
				CreatedAt:      time.Now(),
			},
		},
	}

	gen := NewGenerator(mock, "ops@adortb.com")
	result, err := gen.Generate(context.Background())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	s := result.Sellers[0]
	if s.Name != "" {
		t.Errorf("confidential seller name should be hidden, got %q", s.Name)
	}
	if s.Domain != "" {
		t.Errorf("confidential seller domain should be hidden, got %q", s.Domain)
	}
	if s.IsConfidential != 1 {
		t.Errorf("is_confidential should be 1")
	}
}

func TestGenerate_SerializesCorrectly(t *testing.T) {
	mock := &mockRepo{
		sellers: []*repo.Seller{
			{SellerID: "s1", Name: "Pub1", Domain: "pub1.com", SellerType: "PUBLISHER", Status: "active", CreatedAt: time.Now()},
		},
	}

	gen := NewGenerator(mock, "ops@adortb.com")
	result, _ := gen.Generate(context.Background())

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := raw["sellers"]; !ok {
		t.Error("sellers field missing")
	}
	if _, ok := raw["version"]; !ok {
		t.Error("version field missing")
	}
	if _, ok := raw["contact_email"]; !ok {
		t.Error("contact_email field missing")
	}
}
