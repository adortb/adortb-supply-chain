package schain

import (
	"testing"
)

func TestBuildDirect(t *testing.T) {
	b := NewBuilder("adortb.com", "adortb-adx")
	sc := b.BuildDirect("news.com", "pub123")

	if sc.Complete != 1 {
		t.Errorf("expected complete=1")
	}
	if sc.Ver != "1.0" {
		t.Errorf("expected ver=1.0, got %q", sc.Ver)
	}
	if len(sc.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(sc.Nodes))
	}
	n := sc.Nodes[0]
	if n.ASI != "news.com" || n.SID != "pub123" || n.HP != 1 {
		t.Errorf("unexpected node: %+v", n)
	}
}

func TestBuildWithSelf(t *testing.T) {
	b := NewBuilder("adortb.com", "adortb-adx")
	sc := b.Build(BuildRequest{
		PublisherASI: "news.com",
		PublisherSID: "pub123",
	}, true)

	if len(sc.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(sc.Nodes))
	}
	last := sc.Nodes[len(sc.Nodes)-1]
	if last.ASI != "adortb.com" {
		t.Errorf("last node ASI should be adortb.com, got %q", last.ASI)
	}
}

func TestBuildMultiHop(t *testing.T) {
	b := NewBuilder("adortb.com", "adortb-adx")
	sc := b.Build(BuildRequest{
		PublisherASI: "publisher.com",
		PublisherSID: "pub-1",
		ExtraNodes: []Node{
			{ASI: "intermediary.com", SID: "int-1", HP: 0},
		},
	}, true)

	if len(sc.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(sc.Nodes))
	}
	if sc.Nodes[0].ASI != "publisher.com" {
		t.Errorf("first node should be publisher")
	}
	if sc.Nodes[1].ASI != "intermediary.com" {
		t.Errorf("second node should be intermediary")
	}
	if sc.Nodes[2].ASI != "adortb.com" {
		t.Errorf("third node should be adx")
	}
}
