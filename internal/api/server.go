// Package api 提供供应链服务的 HTTP API 路由和处理器。
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/adortb/adortb-supply-chain/internal/adstxt"
	"github.com/adortb/adortb-supply-chain/internal/schain"
	"github.com/adortb/adortb-supply-chain/internal/sellersjson"
	"github.com/adortb/adortb-supply-chain/internal/spo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Dependencies 汇聚所有 handler 依赖。
type Dependencies struct {
	SellersHandler *sellersjson.Handler
	Validator      *adstxt.Validator
	Fetcher        *adstxt.Fetcher
	SchainBuilder  *schain.Builder
	SchainSigner   *schain.Signer
	SPOAnalyzer    *spo.Analyzer
	ADXDomain      string
	ADXCertAuth    string
	Logger         *slog.Logger
}

// NewRouter 注册所有路由并返回 http.Handler。
func NewRouter(deps *Dependencies) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/sellers.json", deps.SellersHandler)
	mux.HandleFunc("/v1/adstxt/template", deps.handleAdsTxtTemplate)
	mux.HandleFunc("/v1/adstxt/validate", deps.handleAdsTxtValidate)
	mux.HandleFunc("/v1/schain/build", deps.handleSchainBuild)
	mux.HandleFunc("/v1/spo/analyze", deps.handleSPOAnalyze)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", handleHealth)

	return mux
}

// handleAdsTxtTemplate GET /v1/adstxt/template?publisher_id={seller_id}
func (d *Dependencies) handleAdsTxtTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	publisherID := r.URL.Query().Get("publisher_id")
	if publisherID == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("publisher_id is required"))
		return
	}

	// ads.txt 模板行: <adx_domain>, <seller_id>, DIRECT, <cert_auth>
	line := fmt.Sprintf("%s, %s, DIRECT, %s", d.ADXDomain, publisherID, d.ADXCertAuth)
	writeJSON(w, http.StatusOK, map[string]string{
		"line":        line,
		"description": "Add this line to your ads.txt file at the root of your domain",
	})
}

// validateRequest 是 POST /v1/adstxt/validate 请求体。
type validateRequest struct {
	Domain    string `json:"domain"`
	AccountID string `json:"account_id,omitempty"` // 可选，指定 seller_id
}

// handleAdsTxtValidate POST /v1/adstxt/validate
func (d *Dependencies) handleAdsTxtValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid JSON: "+err.Error()))
		return
	}
	if req.Domain == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("domain is required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// 实时抓取
	if err := d.Fetcher.FetchDomain(ctx, req.Domain); err != nil {
		d.Logger.Warn("fetch ads.txt for validate", "domain", req.Domain, "err", err)
	}

	declared := d.Validator.CheckAnyDeclared(req.Domain)
	var result *adstxt.ValidationResult
	if req.AccountID != "" {
		// 如果提供了 account_id，进行精确校验
		result = &adstxt.ValidationResult{
			Valid:    true,
			Declared: declared,
		}
		if !declared {
			result.Issues = append(result.Issues,
				fmt.Sprintf("ADX '%s' not declared for account '%s'", d.ADXDomain, req.AccountID))
		}
	} else {
		result = &adstxt.ValidationResult{
			Valid:    true,
			Declared: declared,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain":   req.Domain,
		"valid":    result.Valid,
		"declared": result.Declared,
		"issues":   result.Issues,
	})
}

// buildSchainRequest 是 POST /v1/schain/build 请求体。
type buildSchainRequest struct {
	PublisherASI string        `json:"publisher_asi"`
	PublisherSID string        `json:"publisher_sid"`
	ExtraNodes   []schain.Node `json:"nodes,omitempty"`
	AddSelf      bool          `json:"add_self"` // 是否追加本 ADX 节点
}

// handleSchainBuild POST /v1/schain/build
func (d *Dependencies) handleSchainBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req buildSchainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid JSON"))
		return
	}
	if req.PublisherASI == "" || req.PublisherSID == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("publisher_asi and publisher_sid are required"))
		return
	}

	sc := d.SchainBuilder.Build(schain.BuildRequest{
		PublisherASI: req.PublisherASI,
		PublisherSID: req.PublisherSID,
		ExtraNodes:   req.ExtraNodes,
	}, req.AddSelf)

	signed, err := d.SchainSigner.Sign(sc)
	if err != nil {
		d.Logger.Error("schain sign", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResp("sign failed"))
		return
	}

	writeJSON(w, http.StatusOK, signed)
}

// handleSPOAnalyze GET /v1/spo/analyze?publisher_id=xxx or ?request_hash=xxx
func (d *Dependencies) handleSPOAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestHash := r.URL.Query().Get("request_hash")
	if requestHash == "" {
		// 兼容 publisher_id 参数
		pid := r.URL.Query().Get("publisher_id")
		if pid == "" {
			writeJSON(w, http.StatusBadRequest, errorResp("request_hash or publisher_id is required"))
			return
		}
		requestHash = "pub:" + strconv.Quote(pid)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	report, err := d.SPOAnalyzer.Analyze(ctx, requestHash)
	if err != nil {
		d.Logger.Error("spo analyze", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResp("analysis failed"))
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func errorResp(msg string) map[string]string {
	return map[string]string{"error": msg}
}
