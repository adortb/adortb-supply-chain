// Package adstxt 实现 IAB ads.txt 标准的解析、校验和爬取。
package adstxt

import (
	"bufio"
	"strings"
)

// Relationship 表示 ads.txt 中授权关系类型。
type Relationship string

const (
	RelationshipDirect   Relationship = "DIRECT"
	RelationshipReseller Relationship = "RESELLER"
)

// Record 表示 ads.txt 中一条有效授权记录。
//
// 格式: <domain>, <account_id>, <relationship>[, <cert_auth_id>]
type Record struct {
	Domain     string       // ADX/SSP 域名
	AccountID  string       // 账号 ID
	Rel        Relationship // DIRECT / RESELLER
	CertAuthID string       // 可选的 TAG 认证 ID
}

// ParseResult 包含解析 ads.txt 的全部结果。
type ParseResult struct {
	Records  []Record
	Errors   []string // 行级错误（不中断解析）
	HasError bool
}

// Parse 解析 ads.txt 文本内容，返回所有有效 Record。
// 遇到格式错误的行记录到 ParseResult.Errors 但不中断解析。
func Parse(content string) *ParseResult {
	result := &ParseResult{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行、注释行、变量行（CONTACT=, SUBDOMAIN=, INVENTORYPARTNERDOMAIN=）
		if line == "" || strings.HasPrefix(line, "#") || isVariableLine(line) {
			continue
		}

		// 去除行内注释
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}

		rec, err := parseLine(line)
		if err != nil {
			result.Errors = append(result.Errors, formatLineError(lineNum, line, err.Error()))
			result.HasError = true
			continue
		}
		result.Records = append(result.Records, rec)
	}

	return result
}

// isVariableLine 判断是否为 ads.txt 变量行（非授权记录）。
func isVariableLine(line string) bool {
	prefixes := []string{"CONTACT=", "SUBDOMAIN=", "INVENTORYPARTNERDOMAIN="}
	upper := strings.ToUpper(line)
	for _, p := range prefixes {
		if strings.HasPrefix(upper, p) {
			return true
		}
	}
	return false
}

// parseLine 解析单行 ads.txt 记录。
func parseLine(line string) (Record, error) {
	// 标准格式: domain, account_id, DIRECT[, cert_auth]
	parts := splitCSV(line)
	if len(parts) < 3 {
		return Record{}, &ParseError{msg: "need at least 3 comma-separated fields"}
	}

	domain := strings.TrimSpace(strings.ToLower(parts[0]))
	accountID := strings.TrimSpace(parts[1])
	rel := strings.TrimSpace(strings.ToUpper(parts[2]))

	if domain == "" {
		return Record{}, &ParseError{msg: "domain is empty"}
	}
	if accountID == "" {
		return Record{}, &ParseError{msg: "account_id is empty"}
	}
	if rel != string(RelationshipDirect) && rel != string(RelationshipReseller) {
		return Record{}, &ParseError{msg: "relationship must be DIRECT or RESELLER, got: " + rel}
	}

	rec := Record{
		Domain:    domain,
		AccountID: accountID,
		Rel:       Relationship(rel),
	}
	if len(parts) >= 4 {
		rec.CertAuthID = strings.TrimSpace(parts[3])
	}
	return rec, nil
}

// splitCSV 按逗号分割，支持多余空格。
func splitCSV(line string) []string {
	parts := strings.Split(line, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func formatLineError(num int, line, msg string) string {
	return "line " + strings.Join([]string{
		"[", string(rune('0'+num/10)), string(rune('0'+num%10)), "] ",
	}, "") + msg + " | " + line
}

// ParseError 是解析行级错误类型。
type ParseError struct{ msg string }

func (e *ParseError) Error() string { return e.msg }
