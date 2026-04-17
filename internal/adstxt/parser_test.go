package adstxt

import (
	"testing"
)

func TestParse_DirectRecord(t *testing.T) {
	content := `adortb.com, pub123, DIRECT, f08c47fec0942fa0`
	result := Parse(content)

	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
	r := result.Records[0]
	if r.Domain != "adortb.com" {
		t.Errorf("domain: got %q, want %q", r.Domain, "adortb.com")
	}
	if r.AccountID != "pub123" {
		t.Errorf("account_id: got %q", r.AccountID)
	}
	if r.Rel != RelationshipDirect {
		t.Errorf("relationship: got %q", r.Rel)
	}
	if r.CertAuthID != "f08c47fec0942fa0" {
		t.Errorf("cert_auth: got %q", r.CertAuthID)
	}
}

func TestParse_ResellerRecord(t *testing.T) {
	content := `google.com, pub-12345, RESELLER`
	result := Parse(content)

	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
	if result.Records[0].Rel != RelationshipReseller {
		t.Errorf("expected RESELLER")
	}
	if result.Records[0].CertAuthID != "" {
		t.Errorf("expected empty cert_auth")
	}
}

func TestParse_SkipsComments(t *testing.T) {
	content := `# This is a comment
adortb.com, pub1, DIRECT
# another comment`
	result := Parse(content)

	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
}

func TestParse_SkipsVariableLines(t *testing.T) {
	content := `CONTACT=ops@example.com
SUBDOMAIN=ads.example.com
adortb.com, pub1, DIRECT
INVENTORYPARTNERDOMAIN=partner.com`
	result := Parse(content)

	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
}

func TestParse_InlineComment(t *testing.T) {
	content := `adortb.com, pub1, DIRECT # inline comment`
	result := Parse(content)

	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
}

func TestParse_InvalidRelationship(t *testing.T) {
	content := `adortb.com, pub1, UNKNOWN`
	result := Parse(content)

	if len(result.Records) != 0 {
		t.Errorf("expected 0 valid records")
	}
	if !result.HasError {
		t.Error("expected HasError=true")
	}
}

func TestParse_MissingFields(t *testing.T) {
	content := `adortb.com, pub1`
	result := Parse(content)

	if len(result.Records) != 0 {
		t.Errorf("expected 0 valid records")
	}
	if !result.HasError {
		t.Error("expected HasError=true")
	}
}

func TestParse_MultipleRecords(t *testing.T) {
	content := `adortb.com, pub1, DIRECT
google.com, pub-999, RESELLER, abc123
openx.com, 5555, RESELLER`
	result := Parse(content)

	if len(result.Records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(result.Records))
	}
}

func TestParse_CaseInsensitiveRelationship(t *testing.T) {
	content := `adortb.com, pub1, direct`
	result := Parse(content)

	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
	if result.Records[0].Rel != RelationshipDirect {
		t.Errorf("expected DIRECT, got %q", result.Records[0].Rel)
	}
}

func TestParse_EmptyContent(t *testing.T) {
	result := Parse("")
	if len(result.Records) != 0 {
		t.Errorf("expected 0 records for empty content")
	}
}
