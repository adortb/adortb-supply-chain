package config

import (
	"fmt"
	"os"
)

// Config 服务配置，从环境变量加载。
type Config struct {
	Port        string
	DatabaseURL string

	// ADX 身份信息，用于 ads.txt 模板和 sellers.json
	ADXDomain   string
	ADXCertAuth string // TAG-ID / cert-auth-id

	// 签名密钥，用于 schain HMAC-SHA256
	SchainSignKey string

	// 爬虫间隔（小时）
	CrawlIntervalHours int
}

// Load 从环境变量加载配置，缺失必填项时返回错误。
func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}

	adxDomain := os.Getenv("ADX_DOMAIN")
	if adxDomain == "" {
		adxDomain = "adortb.com"
	}

	certAuth := os.Getenv("ADX_CERT_AUTH")
	if certAuth == "" {
		certAuth = "f08c47fec0942fa0"
	}

	signKey := os.Getenv("SCHAIN_SIGN_KEY")
	if signKey == "" {
		signKey = "change-me-in-production"
	}

	crawlInterval := 24
	if v := os.Getenv("CRAWL_INTERVAL_HOURS"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &crawlInterval); err != nil {
			crawlInterval = 24
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8091"
	}

	return &Config{
		Port:               port,
		DatabaseURL:        dbURL,
		ADXDomain:          adxDomain,
		ADXCertAuth:        certAuth,
		SchainSignKey:      signKey,
		CrawlIntervalHours: crawlInterval,
	}, nil
}
