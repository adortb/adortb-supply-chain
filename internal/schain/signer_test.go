package schain

import (
	"testing"
)

func TestSignAndVerify(t *testing.T) {
	signer := NewSigner("test-secret-key")
	b := NewBuilder("adortb.com", "adx-1")
	sc := b.BuildDirect("example.com", "pub-99")

	signed, err := signer.Sign(sc)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if signed.Signature == "" {
		t.Error("signature should not be empty")
	}

	ok, err := signer.Verify(sc, signed.Signature)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Error("signature verification failed")
	}
}

func TestVerify_TamperedNodes(t *testing.T) {
	signer := NewSigner("test-secret")
	b := NewBuilder("adortb.com", "adx-1")
	sc := b.BuildDirect("example.com", "pub-1")

	signed, _ := signer.Sign(sc)

	// 篡改节点
	tampered := *sc
	tampered.Nodes = append(tampered.Nodes, Node{ASI: "evil.com", SID: "hack", HP: 0})

	ok, err := signer.Verify(&tampered, signed.Signature)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Error("tampered schain should fail verification")
	}
}

func TestSign_DifferentKeys(t *testing.T) {
	b := NewBuilder("adortb.com", "adx-1")
	sc := b.BuildDirect("example.com", "pub-1")

	signer1 := NewSigner("key-1")
	signer2 := NewSigner("key-2")

	signed, _ := signer1.Sign(sc)

	ok, _ := signer2.Verify(sc, signed.Signature)
	if ok {
		t.Error("should not verify with different key")
	}
}

func TestToJSON(t *testing.T) {
	b := NewBuilder("adortb.com", "adx-1")
	sc := b.BuildDirect("news.com", "pub-10")

	data, err := ToJSON(sc)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	if len(data) == 0 {
		t.Error("JSON output should not be empty")
	}
}
