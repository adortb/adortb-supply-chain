package sellersjson

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const cacheTTL = 5 * time.Minute

// Handler 提供 GET /sellers.json HTTP 接口，带内存缓存避免每次查数据库。
type Handler struct {
	gen    *Generator
	logger *slog.Logger

	mu          sync.RWMutex
	cachedBytes []byte
	cachedAt    time.Time
}

// NewHandler 创建 sellers.json HTTP handler。
func NewHandler(gen *Generator, logger *slog.Logger) *Handler {
	return &Handler{gen: gen, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := h.getCached(r.Context())
	if err != nil {
		h.logger.Error("sellers.json generate failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) getCached(ctx context.Context) ([]byte, error) {
	h.mu.RLock()
	if time.Since(h.cachedAt) < cacheTTL && h.cachedBytes != nil {
		data := h.cachedBytes
		h.mu.RUnlock()
		return data, nil
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	// double-check after acquiring write lock
	if time.Since(h.cachedAt) < cacheTTL && h.cachedBytes != nil {
		return h.cachedBytes, nil
	}

	result, err := h.gen.Generate(ctx)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	h.cachedBytes = data
	h.cachedAt = time.Now()
	return data, nil
}

// Invalidate 主动清除缓存（seller 数据变更时调用）。
func (h *Handler) Invalidate() {
	h.mu.Lock()
	h.cachedBytes = nil
	h.mu.Unlock()
}
