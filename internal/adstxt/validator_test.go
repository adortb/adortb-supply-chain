package adstxt

import "testing"

func TestValidator_Declared(t *testing.T) {
	v := NewValidator("adortb.com", "f08c47fec0942fa0")
	content := `adortb.com, pub123, DIRECT, f08c47fec0942fa0`
	result := v.ValidateContent(content, "pub123")

	if !result.Declared {
		t.Error("expected declared=true")
	}
	if !result.Valid {
		t.Errorf("expected valid=true, issues: %v", result.Issues)
	}
}

func TestValidator_NotDeclared(t *testing.T) {
	v := NewValidator("adortb.com", "f08c47fec0942fa0")
	content := `google.com, pub-123, RESELLER`
	result := v.ValidateContent(content, "pub123")

	if result.Declared {
		t.Error("expected declared=false")
	}
	if len(result.Issues) == 0 {
		t.Error("expected issues for undeclared ADX")
	}
}

func TestValidator_CertAuthMismatch(t *testing.T) {
	v := NewValidator("adortb.com", "f08c47fec0942fa0")
	content := `adortb.com, pub123, DIRECT, wrongcertauth`
	result := v.ValidateContent(content, "pub123")

	if !result.Declared {
		t.Error("expected declared=true even with cert mismatch")
	}
	// cert mismatch 应产生 issue
	if len(result.Issues) == 0 {
		t.Error("expected cert_auth mismatch issue")
	}
}

func TestValidator_CheckAnyDeclared(t *testing.T) {
	v := NewValidator("adortb.com", "")

	tests := []struct {
		content  string
		expected bool
	}{
		{"adortb.com, pub1, DIRECT", true},
		{"google.com, pub-1, RESELLER", false},
		{"ADORTB.COM, pub2, RESELLER", true}, // case insensitive
		{"", false},
	}

	for _, tc := range tests {
		got := v.CheckAnyDeclared(tc.content)
		if got != tc.expected {
			t.Errorf("content=%q: got %v, want %v", tc.content, got, tc.expected)
		}
	}
}
