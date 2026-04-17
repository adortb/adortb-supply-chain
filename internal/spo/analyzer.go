// Package spo 实现 Supply Path Optimization（供应路径优化）分析。
package spo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/adortb/adortb-supply-chain/internal/repo"
	"github.com/adortb/adortb-supply-chain/internal/schain"
)

// PathSummary 描述单条供应路径的分析结果。
type PathSummary struct {
	RequestHash  string
	Depth        int
	QualityScore float64
	Nodes        []schain.Node
	IsDuplicate  bool // 与其他路径存在重叠
}

// Report 是 SPO 分析报告。
type Report struct {
	PublisherID    int64
	TotalPaths     int
	DuplicatePaths int
	AvgDepth       float64
	AvgQuality     float64
	Paths          []PathSummary
	Issues         []string
}

// Analyzer 分析 publisher 的供应路径，识别重复路径并给出质量评分。
type Analyzer struct {
	repo repo.Repository
}

// NewAnalyzer 创建 Analyzer。
func NewAnalyzer(r repo.Repository) *Analyzer {
	return &Analyzer{repo: r}
}

// Analyze 对指定 requestHash 下的所有供应路径进行分析。
func (a *Analyzer) Analyze(ctx context.Context, requestHash string) (*Report, error) {
	paths, err := a.repo.ListSupplyPaths(ctx, requestHash)
	if err != nil {
		return nil, fmt.Errorf("spo: list paths: %w", err)
	}

	report := &Report{
		TotalPaths: len(paths),
	}

	summaries := make([]PathSummary, 0, len(paths))
	nodeFingerprints := make(map[string]int) // fingerprint → first path index

	totalDepth := 0.0
	totalQuality := 0.0

	for _, p := range paths {
		var nodes []schain.Node
		if err := json.Unmarshal(p.Path, &nodes); err != nil {
			report.Issues = append(report.Issues,
				fmt.Sprintf("path %d: unmarshal nodes: %v", p.ID, err))
			continue
		}

		fp := fingerprint(nodes)
		isDup := false
		if _, seen := nodeFingerprints[fp]; seen {
			isDup = true
			report.DuplicatePaths++
		} else {
			nodeFingerprints[fp] = len(summaries)
		}

		totalDepth += float64(p.Depth)
		totalQuality += p.QualityScore

		summaries = append(summaries, PathSummary{
			RequestHash:  p.RequestHash,
			Depth:        p.Depth,
			QualityScore: p.QualityScore,
			Nodes:        nodes,
			IsDuplicate:  isDup,
		})
	}

	if len(summaries) > 0 {
		report.AvgDepth = totalDepth / float64(len(summaries))
		report.AvgQuality = totalQuality / float64(len(summaries))
	}

	report.Paths = summaries

	if report.DuplicatePaths > 0 {
		report.Issues = append(report.Issues,
			fmt.Sprintf("%d duplicate supply paths detected — consider deduplication", report.DuplicatePaths))
	}
	if report.AvgDepth > 3 {
		report.Issues = append(report.Issues,
			"average path depth exceeds 3 hops — consider preferring shorter paths")
	}

	return report, nil
}

// ScorePath 根据节点深度和节点质量计算路径质量分（0~1）。
// 越短越直接的路径分越高。
func ScorePath(nodes []schain.Node) float64 {
	if len(nodes) == 0 {
		return 0
	}
	// 深度惩罚：每增加一个节点降低 0.1 分，最低 0.1
	score := 1.0 - float64(len(nodes)-1)*0.1
	if score < 0.1 {
		score = 0.1
	}
	// DIRECT 节点加分
	for _, n := range nodes {
		if n.HP == 1 {
			score += 0.05
		}
	}
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// fingerprint 生成节点路径的唯一指纹（ASI:SID 拼接）。
func fingerprint(nodes []schain.Node) string {
	parts := make([]string, len(nodes))
	for i, n := range nodes {
		parts[i] = n.ASI + ":" + n.SID
	}
	return strings.Join(parts, "|")
}
