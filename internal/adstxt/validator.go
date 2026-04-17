package adstxt

import "strings"

// ValidationResult 描述 publisher ads.txt 校验结果。
type ValidationResult struct {
	Valid    bool
	Declared bool   // 是否声明了本 ADX
	Issues   []string
}

// Validator 校验 publisher 域名的 ads.txt 是否正确声明了本 ADX。
type Validator struct {
	adxDomain   string
	adxCertAuth string
}

// NewValidator 创建校验器，adxDomain 为本 ADX 域名，certAuth 为 TAG-ID。
func NewValidator(adxDomain, certAuth string) *Validator {
	return &Validator{
		adxDomain:   strings.ToLower(adxDomain),
		adxCertAuth: strings.ToLower(certAuth),
	}
}

// ValidateContent 校验给定的 ads.txt 原始内容是否声明了本 ADX。
// publisherAccountID 为媒体方在本 ADX 的 account_id（来自 sellers 表 seller_id）。
func (v *Validator) ValidateContent(content, publisherAccountID string) *ValidationResult {
	result := &ValidationResult{Valid: true}
	parsed := Parse(content)

	if parsed.HasError {
		result.Valid = false
		result.Issues = append(result.Issues, "ads.txt contains parse errors")
	}

	for _, rec := range parsed.Records {
		if strings.EqualFold(rec.Domain, v.adxDomain) &&
			strings.EqualFold(rec.AccountID, publisherAccountID) {
			result.Declared = true

			// 检查 cert_auth 是否匹配（可选项，不影响 declared 状态）
			if v.adxCertAuth != "" && rec.CertAuthID != "" &&
				!strings.EqualFold(rec.CertAuthID, v.adxCertAuth) {
				result.Issues = append(result.Issues,
					"cert_auth_id mismatch: expected "+v.adxCertAuth+", got "+rec.CertAuthID)
			}
			return result
		}
	}

	result.Declared = false
	result.Issues = append(result.Issues,
		"ADX '"+v.adxDomain+"' not declared for account '"+publisherAccountID+"'")
	return result
}

// CheckAnyDeclared 仅检查内容中是否存在任意声明本 ADX 域名的记录（不限 account_id）。
func (v *Validator) CheckAnyDeclared(content string) bool {
	parsed := Parse(content)
	for _, rec := range parsed.Records {
		if strings.EqualFold(rec.Domain, v.adxDomain) {
			return true
		}
	}
	return false
}
