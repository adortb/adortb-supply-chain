package spo

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/adortb/adortb-supply-chain/internal/repo"
	"github.com/adortb/adortb-supply-chain/internal/schain"
)

type mockSPORepo struct {
	paths []*repo.SupplyPath
}

func (m *mockSPORepo) ListActiveSellers(_ context.Context) ([]*repo.Seller, error) { return nil, nil }
func (m *mockSPORepo) GetSellerByID(_ context.Context, _ string) (*repo.Seller, error) {
	return nil, nil
}
func (m *mockSPORepo) UpsertAdsTxtRecord(_ context.Context, _ *repo.AdsTxtRecord) error { return nil }
func (m *mockSPORepo) GetAdsTxtRecord(_ context.Context, _ string) (*repo.AdsTxtRecord, error) {
	return nil, nil
}
func (m *mockSPORepo) ListUndeclaredDomains(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockSPORepo) InsertSupplyPath(_ context.Context, _ *repo.SupplyPath) error { return nil }
func (m *mockSPORepo) ListSupplyPaths(_ context.Context, _ string) ([]*repo.SupplyPath, error) {
	return m.paths, nil
}

func marshalNodes(nodes []schain.Node) json.RawMessage {
	b, _ := json.Marshal(nodes)
	return b
}

func TestAnalyze_NoDuplicates(t *testing.T) {
	mock := &mockSPORepo{
		paths: []*repo.SupplyPath{
			{
				ID:          1,
				RequestHash: "hash1",
				Path:        marshalNodes([]schain.Node{{ASI: "pub.com", SID: "p1", HP: 1}}),
				Depth:       1, QualityScore: 0.9, SeenAt: time.Now(),
			},
			{
				ID:          2,
				RequestHash: "hash1",
				Path:        marshalNodes([]schain.Node{{ASI: "other.com", SID: "p2", HP: 1}}),
				Depth:       1, QualityScore: 0.85, SeenAt: time.Now(),
			},
		},
	}

	analyzer := NewAnalyzer(mock)
	report, err := analyzer.Analyze(context.Background(), "hash1")
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}

	if report.DuplicatePaths != 0 {
		t.Errorf("expected 0 duplicates, got %d", report.DuplicatePaths)
	}
	if report.TotalPaths != 2 {
		t.Errorf("expected 2 total, got %d", report.TotalPaths)
	}
}

func TestAnalyze_DetectsDuplicates(t *testing.T) {
	nodes := []schain.Node{{ASI: "pub.com", SID: "p1", HP: 1}}
	mock := &mockSPORepo{
		paths: []*repo.SupplyPath{
			{ID: 1, RequestHash: "h", Path: marshalNodes(nodes), Depth: 1, QualityScore: 0.9, SeenAt: time.Now()},
			{ID: 2, RequestHash: "h", Path: marshalNodes(nodes), Depth: 1, QualityScore: 0.8, SeenAt: time.Now()},
		},
	}

	analyzer := NewAnalyzer(mock)
	report, _ := analyzer.Analyze(context.Background(), "h")

	if report.DuplicatePaths != 1 {
		t.Errorf("expected 1 duplicate, got %d", report.DuplicatePaths)
	}
}

func TestScorePath(t *testing.T) {
	tests := []struct {
		nodes    []schain.Node
		minScore float64
	}{
		{[]schain.Node{{ASI: "a", SID: "1", HP: 1}}, 0.8},                                                         // 1 node
		{[]schain.Node{{ASI: "a", SID: "1", HP: 1}, {ASI: "b", SID: "2", HP: 0}}, 0.5},                            // 2 nodes
		{[]schain.Node{{ASI: "a", SID: "1"}, {ASI: "b", SID: "2"}, {ASI: "c", SID: "3"}, {ASI: "d", SID: "4"}}, 0.1}, // deep
		{nil, 0},
	}

	for _, tc := range tests {
		score := ScorePath(tc.nodes)
		if score < tc.minScore {
			t.Errorf("nodes len=%d: score=%f, want >= %f", len(tc.nodes), score, tc.minScore)
		}
		if score > 1.0 {
			t.Errorf("score %f exceeds 1.0", score)
		}
	}
}

func TestDeduplicatedPaths(t *testing.T) {
	paths := []PathSummary{
		{Nodes: []schain.Node{{ASI: "a", SID: "1"}}, QualityScore: 0.8},
		{Nodes: []schain.Node{{ASI: "a", SID: "1"}}, QualityScore: 0.9}, // dup, higher score
		{Nodes: []schain.Node{{ASI: "b", SID: "2"}}, QualityScore: 0.7},
	}

	result := DeduplicatedPaths(paths)
	if len(result) != 2 {
		t.Fatalf("expected 2 unique paths, got %d", len(result))
	}
}
