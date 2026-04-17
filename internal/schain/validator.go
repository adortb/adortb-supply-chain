package schain

import "fmt"

// ValidateResult 描述 SupplyChain 验证结果。
type ValidateResult struct {
	Valid    bool
	Complete bool
	Issues   []string
}

// Validate 校验 SupplyChain 对象的完整性和合规性。
// complete=1 且至少有一个节点才视为有效。
func Validate(sc *SupplyChain) *ValidateResult {
	result := &ValidateResult{Valid: true}

	if sc == nil {
		result.Valid = false
		result.Issues = append(result.Issues, "schain is nil")
		return result
	}

	if sc.Ver != "1.0" {
		result.Issues = append(result.Issues,
			fmt.Sprintf("unsupported version: %q (expected 1.0)", sc.Ver))
	}

	result.Complete = sc.Complete == 1
	if !result.Complete {
		result.Issues = append(result.Issues, "complete=0: supply chain is not complete")
	}

	if len(sc.Nodes) == 0 {
		result.Valid = false
		result.Issues = append(result.Issues, "schain has no nodes")
		return result
	}

	for i, n := range sc.Nodes {
		if n.ASI == "" {
			result.Issues = append(result.Issues,
				fmt.Sprintf("node[%d]: asi is empty", i))
			result.Valid = false
		}
		if n.SID == "" {
			result.Issues = append(result.Issues,
				fmt.Sprintf("node[%d]: sid is empty", i))
			result.Valid = false
		}
		if n.HP != 0 && n.HP != 1 {
			result.Issues = append(result.Issues,
				fmt.Sprintf("node[%d]: hp must be 0 or 1, got %d", i, n.HP))
			result.Valid = false
		}
	}

	if len(result.Issues) > 0 && result.Valid {
		// 有 issue 但未标 invalid 的情况，降低可信度但保持 valid
		result.Valid = result.Complete && len(sc.Nodes) > 0
	}

	return result
}
