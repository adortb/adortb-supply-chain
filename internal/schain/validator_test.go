package schain

import "testing"

func TestValidate_ValidChain(t *testing.T) {
	sc := &SupplyChain{
		Complete: 1,
		Ver:      "1.0",
		Nodes: []Node{
			{ASI: "publisher.com", SID: "pub1", HP: 1},
			{ASI: "adortb.com", SID: "adx1", HP: 1},
		},
	}
	r := Validate(sc)
	if !r.Valid {
		t.Errorf("expected valid, issues: %v", r.Issues)
	}
	if !r.Complete {
		t.Error("expected complete=true")
	}
}

func TestValidate_NilSchain(t *testing.T) {
	r := Validate(nil)
	if r.Valid {
		t.Error("expected invalid for nil schain")
	}
}

func TestValidate_NoNodes(t *testing.T) {
	sc := &SupplyChain{Complete: 1, Ver: "1.0", Nodes: nil}
	r := Validate(sc)
	if r.Valid {
		t.Error("expected invalid for empty nodes")
	}
}

func TestValidate_Incomplete(t *testing.T) {
	sc := &SupplyChain{
		Complete: 0,
		Ver:      "1.0",
		Nodes:    []Node{{ASI: "pub.com", SID: "p1", HP: 1}},
	}
	r := Validate(sc)
	if r.Complete {
		t.Error("expected complete=false")
	}
	if len(r.Issues) == 0 {
		t.Error("expected issues for incomplete chain")
	}
}

func TestValidate_EmptyASI(t *testing.T) {
	sc := &SupplyChain{
		Complete: 1,
		Ver:      "1.0",
		Nodes:    []Node{{ASI: "", SID: "p1", HP: 1}},
	}
	r := Validate(sc)
	if r.Valid {
		t.Error("expected invalid for empty ASI")
	}
}

func TestValidate_InvalidHP(t *testing.T) {
	sc := &SupplyChain{
		Complete: 1,
		Ver:      "1.0",
		Nodes:    []Node{{ASI: "pub.com", SID: "p1", HP: 2}},
	}
	r := Validate(sc)
	if r.Valid {
		t.Error("expected invalid for HP=2")
	}
}

func TestSigner_SignAndVerify(t *testing.T) {
	s := NewSigner("test-secret-key")
	sc := &SupplyChain{
		Complete: 1,
		Ver:      "1.0",
		Nodes:    []Node{{ASI: "pub.com", SID: "p1", HP: 1}},
	}
	signed, err := s.Sign(sc)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}
	if signed.Signature == "" {
		t.Error("expected non-empty signature")
	}
	ok, err := s.Verify(sc, signed.Signature)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if !ok {
		t.Error("expected signature to verify correctly")
	}
}

func TestSigner_WrongKey(t *testing.T) {
	s1 := NewSigner("key-one")
	s2 := NewSigner("key-two")
	sc := &SupplyChain{
		Complete: 1,
		Ver:      "1.0",
		Nodes:    []Node{{ASI: "pub.com", SID: "p1", HP: 1}},
	}
	signed, _ := s1.Sign(sc)
	ok, _ := s2.Verify(sc, signed.Signature)
	if ok {
		t.Error("expected verification to fail with different key")
	}
}
