package schain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// SignedSupplyChain 在 SupplyChain 基础上附加 HMAC-SHA256 签名。
type SignedSupplyChain struct {
	*SupplyChain
	Signature string `json:"ext_sig,omitempty"`
}

// Signer 对 SupplyChain 进行 HMAC-SHA256 签名，防止中间链路篡改。
type Signer struct {
	key []byte
}

// NewSigner 创建 Signer，key 为 HMAC 密钥。
func NewSigner(key string) *Signer {
	return &Signer{key: []byte(key)}
}

// Sign 对 SupplyChain 的规范序列化字符串进行签名，返回 SignedSupplyChain。
func (s *Signer) Sign(sc *SupplyChain) (*SignedSupplyChain, error) {
	canonical, err := canonicalize(sc)
	if err != nil {
		return nil, fmt.Errorf("schain signer: canonicalize: %w", err)
	}

	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(canonical))
	sig := hex.EncodeToString(mac.Sum(nil))

	return &SignedSupplyChain{
		SupplyChain: sc,
		Signature:   sig,
	}, nil
}

// Verify 验证签名是否匹配。
func (s *Signer) Verify(sc *SupplyChain, sig string) (bool, error) {
	canonical, err := canonicalize(sc)
	if err != nil {
		return false, fmt.Errorf("schain signer: canonicalize: %w", err)
	}

	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(canonical))
	expected := mac.Sum(nil)

	decoded, err := hex.DecodeString(sig)
	if err != nil {
		return false, nil
	}
	return hmac.Equal(expected, decoded), nil
}

// canonicalize 生成确定性序列化字符串：ver|complete|asi:sid:hp,asi:sid:hp,...
func canonicalize(sc *SupplyChain) (string, error) {
	if sc == nil {
		return "", fmt.Errorf("nil schain")
	}
	nodes := make([]string, 0, len(sc.Nodes))
	for _, n := range sc.Nodes {
		nodes = append(nodes, fmt.Sprintf("%s:%s:%d", n.ASI, n.SID, n.HP))
	}
	return fmt.Sprintf("%s|%d|%s", sc.Ver, sc.Complete, strings.Join(nodes, ",")), nil
}

// ToJSON 将 SupplyChain 序列化为 JSON bytes（用于注入 BidRequest.Source.Ext）。
func ToJSON(sc *SupplyChain) ([]byte, error) {
	return json.Marshal(sc)
}
