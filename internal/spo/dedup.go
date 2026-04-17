package spo

import (
	"github.com/adortb/adortb-supply-chain/internal/schain"
)

// DeduplicatedPaths 对多条供应路径去重，保留质量分最高的唯一路径。
// 路径以节点指纹（ASI:SID 序列）作为唯一键。
func DeduplicatedPaths(paths []PathSummary) []PathSummary {
	best := make(map[string]*PathSummary, len(paths))

	for i := range paths {
		p := &paths[i]
		fp := fingerprint(p.Nodes)
		if existing, ok := best[fp]; !ok || p.QualityScore > existing.QualityScore {
			best[fp] = p
		}
	}

	result := make([]PathSummary, 0, len(best))
	for _, p := range best {
		result = append(result, *p)
	}
	return result
}

// FilterByMinQuality 过滤质量分低于阈值的路径。
func FilterByMinQuality(paths []PathSummary, minScore float64) []PathSummary {
	result := make([]PathSummary, 0, len(paths))
	for _, p := range paths {
		if p.QualityScore >= minScore {
			result = append(result, p)
		}
	}
	return result
}

// PreferDirectPaths 优先返回含 HP=1 节点（直接路径）的供应链。
func PreferDirectPaths(chains []*schain.SupplyChain) []*schain.SupplyChain {
	direct := make([]*schain.SupplyChain, 0, len(chains))
	indirect := make([]*schain.SupplyChain, 0)

	for _, sc := range chains {
		if isDirectChain(sc) {
			direct = append(direct, sc)
		} else {
			indirect = append(indirect, sc)
		}
	}

	return append(direct, indirect...)
}

func isDirectChain(sc *schain.SupplyChain) bool {
	for _, n := range sc.Nodes {
		if n.HP == 1 {
			return true
		}
	}
	return false
}
