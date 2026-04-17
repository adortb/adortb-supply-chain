// 供应链透明度服务入口：sellers.json / ads.txt 校验 / SupplyChain Object / SPO 分析
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adortb/adortb-supply-chain/internal/adstxt"
	"github.com/adortb/adortb-supply-chain/internal/api"
	"github.com/adortb/adortb-supply-chain/internal/config"
	"github.com/adortb/adortb-supply-chain/internal/metrics"
	"github.com/adortb/adortb-supply-chain/internal/repo"
	"github.com/adortb/adortb-supply-chain/internal/schain"
	"github.com/adortb/adortb-supply-chain/internal/sellersjson"
	"github.com/adortb/adortb-supply-chain/internal/spo"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	metrics.Register()

	pg, err := repo.New(cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect db", "error", err)
		os.Exit(1)
	}
	defer pg.Close()

	validator := adstxt.NewValidator(cfg.ADXDomain, cfg.ADXCertAuth)
	fetcher := adstxt.NewFetcher(pg, validator, cfg.ADXDomain, logger)
	generator := sellersjson.NewGenerator(pg, "ops@adortb.com")
	sellersHandler := sellersjson.NewHandler(generator, logger)
	schainBuilder := schain.NewBuilder(cfg.ADXDomain, "adortb-adx")
	schainSigner := schain.NewSigner(cfg.SchainSignKey)
	spoAnalyzer := spo.NewAnalyzer(pg)

	// 启动定时爬虫
	go startCrawler(logger, pg, fetcher, cfg.CrawlIntervalHours)

	router := api.NewRouter(&api.Dependencies{
		SellersHandler: sellersHandler,
		Validator:      validator,
		Fetcher:        fetcher,
		SchainBuilder:  schainBuilder,
		SchainSigner:   schainSigner,
		SPOAnalyzer:    spoAnalyzer,
		ADXDomain:      cfg.ADXDomain,
		ADXCertAuth:    cfg.ADXCertAuth,
		Logger:         logger,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("supply-chain service starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
}

func startCrawler(logger *slog.Logger, pg *repo.PGRepo, fetcher *adstxt.Fetcher, intervalHours int) {
	ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
	defer ticker.Stop()

	// 启动时立即执行一次
	crawlOnce(logger, pg, fetcher)

	for range ticker.C {
		crawlOnce(logger, pg, fetcher)
	}
}

func crawlOnce(logger *slog.Logger, pg *repo.PGRepo, fetcher *adstxt.Fetcher) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	domains, err := pg.ListUndeclaredDomains(ctx)
	if err != nil {
		logger.Error("list undeclared domains", "error", err)
		return
	}
	logger.Info("crawling ads.txt", "domains", len(domains))
	fetcher.FetchAll(ctx, domains)
}
