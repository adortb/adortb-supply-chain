// Package schain 实现 OpenRTB SupplyChain Object 构造（IAB OpenRTB 2.6 / SupplyChain 1.0）。
package schain

// Node 表示供应链中的一个参与节点。
type Node struct {
	// ASI: 授权卖方标识符系统，即 domain
	ASI string `json:"asi"`
	// SID: 该系统内的账号 ID
	SID string `json:"sid"`
	// HP: 1 = 该节点直接参与最终展示决策
	HP int `json:"hp"`
	// RID: 请求 ID（可选）
	RID string `json:"rid,omitempty"`
	// Name: 卖方名称（可选）
	Name string `json:"name,omitempty"`
	// Domain: 卖方域名（可选，非 ASI）
	Domain string `json:"domain,omitempty"`
}

// SupplyChain 是 OpenRTB source.ext.schain 对象。
type SupplyChain struct {
	Complete int    `json:"complete"` // 1 = 链路完整
	Nodes    []Node `json:"nodes"`
	Ver      string `json:"ver"` // 始终为 "1.0"
}

// BuildRequest 是构造 schain 的输入参数。
type BuildRequest struct {
	PublisherASI string // 媒体方 ASI（域名）
	PublisherSID string // 媒体方在本 ADX 的 seller_id
	ExtraNodes   []Node // 中间路由节点（reseller 场景）

	// ADX 自身节点信息
	ADXASI string
	ADXSID string
}

// Builder 负责构造 SupplyChain 对象。
type Builder struct {
	adxASI string
	adxSID string
}

// NewBuilder 创建 Builder，adxASI 为 ADX 域名，adxSID 为 ADX 的账户 ID。
func NewBuilder(adxASI, adxSID string) *Builder {
	return &Builder{adxASI: adxASI, adxSID: adxSID}
}

// Build 根据请求构造 SupplyChain。
// 若 addSelf=true，则在链路尾部追加本 ADX 作为 intermediary node。
func (b *Builder) Build(req BuildRequest, addSelf bool) *SupplyChain {
	nodes := make([]Node, 0, len(req.ExtraNodes)+2)

	// publisher 节点总是第一个
	nodes = append(nodes, Node{
		ASI: req.PublisherASI,
		SID: req.PublisherSID,
		HP:  1,
	})

	// 中间节点（reseller 场景）
	nodes = append(nodes, req.ExtraNodes...)

	// 本 ADX 作为最后一跳（reseller 场景下追加）
	if addSelf {
		asi := b.adxASI
		sid := b.adxSID
		if req.ADXASI != "" {
			asi = req.ADXASI
		}
		if req.ADXSID != "" {
			sid = req.ADXSID
		}
		nodes = append(nodes, Node{
			ASI: asi,
			SID: sid,
			HP:  1,
		})
	}

	return &SupplyChain{
		Complete: 1,
		Nodes:    nodes,
		Ver:      "1.0",
	}
}

// BuildDirect 构造最简供应链：仅 publisher 一个节点（DIRECT 场景）。
func (b *Builder) BuildDirect(publisherASI, publisherSID string) *SupplyChain {
	return b.Build(BuildRequest{
		PublisherASI: publisherASI,
		PublisherSID: publisherSID,
	}, false)
}
