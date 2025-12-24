package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"service-currency/internal"
	"service-currency/internal/api/http/middleware"
	"service-currency/internal/currency_freaks"
	"service-currency/internal/postgresql"
	"service-currency/internal/postgresql/migrations"
	"syscall"
	"time"

	rateshttp "service-currency/internal/api/http/rates"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	// env
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("не удалось загрузить конфиг: %w", err)
	}

	// DB
	dbCtx, cancelDB := context.WithTimeout(ctx, 5*time.Second)
	defer cancelDB()

	pool, err := pgxpool.New(dbCtx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к БД: %w", err)
	}
	defer pool.Close()

	// storage + migrations
	storage := postgresql.NewCurrencyStorage(pool)
	apiKeyStorage := postgresql.NewAPIKeyStorage(pool)
	migrator := migrations.New(pool)
	err = migrator.Setup(dbCtx)
	if err != nil {
		return fmt.Errorf("ensure tables: %w", err)
	}

	// client
	var client currencyFreaks.RatesClient = currencyFreaks.New(cfg.APIKey, storage)

	// instant fetch
	resp, err := client.FetchAndSaveLatest(ctx, storage, cfg.BaseCCY, cfg.Symbols)
	if err != nil {
		return fmt.Errorf("fetch latest: %w", err)
	}

	log.Printf("rates updated (base=%s date=%s)", resp.Base, resp.Date)

	// cron
	loc, err := time.LoadLocation(cfg.Location)
	if err != nil {
		return fmt.Errorf("load location %s: %w", cfg.Location, err)
	}

	scheduler := cron.New(
		cron.WithLocation(loc),
		cron.WithParser(cron.NewParser(cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow)),
	)

	// logger
	reqAuditStorage := postgresql.NewRequestLogStorage(pool)
	reqAuditLogger := internal.NewStorageAuditLogger(reqAuditStorage)

	// HTTP handler
	ratesService := internal.NewRateConverter(storage)
	ratesHandler := rateshttp.New(ratesService, reqAuditLogger)

	mux := http.NewServeMux()

	// Middleware
	apiKeyValidator := internal.NewAPIKeyValidator(apiKeyStorage, cfg.EncodingKey)
	authMiddleware := middleware.APIKeyAuth(apiKeyValidator)
	mw := []func(next http.Handler) http.Handler{authMiddleware}
	ratesHandler.Register(mux)

	ewg, gctx := errgroup.WithContext(ctx)

	_, err = scheduler.AddFunc(cfg.CronSpec, func() {
		resp, err := client.FetchAndSaveLatest(gctx, storage, cfg.BaseCCY, cfg.Symbols)
		if err != nil {
			log.Printf("scheduled job failed: %v", err)
		} else {
			log.Printf("rates updated: base=%s date=%s", resp.Base, resp.Date)
		}
	})
	if err != nil {
		return fmt.Errorf("add cron func: %w", err)
	}

	ewg.Go(func() error {
		return runCron(gctx, scheduler)
	})

	ewg.Go(func() error {
		return serveHTTP(gctx, ":"+cfg.HTTPPort, mux, mw)
	})

	log.Println("Running. Stop with Ctrl+C / SIGTERM.")
	return ewg.Wait()
}

// Не могу убрать возврат ошибки, из-за требования метода ewg.Go
func runCron(ctx context.Context, c *cron.Cron) error {
	c.Start()
	defer func() {
		stopCtx := c.Stop()
		<-stopCtx.Done()
	}()

	<-ctx.Done()
	return nil
}

func serveHTTP(ctx context.Context, addr string, h http.Handler, mws []func(http.Handler) http.Handler) error {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}

	srv := &http.Server{Addr: addr, Handler: h}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	log.Printf("HTTP listening on %s", addr)
	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
