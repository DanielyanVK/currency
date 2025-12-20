package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"service-currency/internal/service/logger"
	"syscall"
	"time"

	rateshttp "service-currency/internal/api/http/rates"
	"service-currency/internal/clients/currencyFreaks"
	"service-currency/internal/repository/migrations"
	"service-currency/internal/repository/postgresql"
	ratessvc "service-currency/internal/service/rates"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
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
	migrator := migrations.New(pool)
	if err := migrator.Setup(dbCtx); err != nil {
		return fmt.Errorf("ensure tables: %w", err)
	}

	// client
	client := currencyFreaks.New(cfg.APIKey, storage)

	// instant fetch
	if resp, err := client.FetchAndSaveLatest(ctx, storage, cfg.BaseCCY, cfg.Symbols); err != nil {
		log.Printf("initial fetch failed: %v", err)
	} else {
		log.Printf("rates updated: base=%s date=%s", resp.Base, resp.Date)
	}

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
	reqLogStorage := postgresql.NewRequestLogStorage(pool)
	reqLogger := logger.New(reqLogStorage)

	// rates HTTP handler
	ratesService := ratessvc.New(storage)
	ratesHandler := rateshttp.New(ratesService, reqLogger)

	mux := http.NewServeMux()
	ratesHandler.Register(mux)

	g, gctx := errgroup.WithContext(ctx)

	_, err = scheduler.AddFunc(cfg.CronSpec, func() {
		if resp, err := client.FetchAndSaveLatest(gctx, storage, cfg.BaseCCY, cfg.Symbols); err != nil {
			log.Printf("scheduled job failed: %v", err)
		} else {
			log.Printf("rates updated: base=%s date=%s", resp.Base, resp.Date)
		}
	})
	if err != nil {
		return fmt.Errorf("add cron func: %w", err)
	}

	g.Go(func() error {
		return runCron(gctx, scheduler)
	})

	g.Go(func() error {
		return serveHTTP(gctx, ":"+cfg.HTTPPort, mux)
	})

	log.Println("Running. Stop with Ctrl+C / SIGTERM.")
	return g.Wait()
}

func runCron(ctx context.Context, c *cron.Cron) error {
	c.Start()
	defer func() {
		stopCtx := c.Stop()
		<-stopCtx.Done()
	}()

	<-ctx.Done()
	return nil
}

func serveHTTP(ctx context.Context, addr string, h http.Handler) error {
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
