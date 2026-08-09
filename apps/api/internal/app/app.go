package app

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agungardiyanta/UrlShorter/apps/api/internal/cache"
	"github.com/agungardiyanta/UrlShorter/apps/api/internal/config"
	"github.com/agungardiyanta/UrlShorter/apps/api/internal/events"
	"github.com/agungardiyanta/UrlShorter/apps/api/internal/httpapi"
	"github.com/agungardiyanta/UrlShorter/apps/api/internal/store"
	"github.com/agungardiyanta/UrlShorter/apps/api/internal/telemetry"
)

type Application struct {
	cfg       config.Config
	store     *store.PostgresStore
	cache     *cache.RedisCache
	publisher *events.KafkaPublisher
	server    *httpapi.Server
	shutdown  func(context.Context) error
}

func New() (*Application, error) {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	shutdownTelemetry, err := telemetry.Init(ctx, cfg)
	if err != nil {
		return nil, err
	}

	st, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		_ = shutdownTelemetry(context.Background())
		return nil, err
	}

	rc, err := cache.NewRedisCache(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.CacheTTL)
	if err != nil {
		st.Close()
		_ = shutdownTelemetry(context.Background())
		return nil, err
	}

	publisher := events.NewKafkaPublisher(cfg.KafkaBrokers, cfg.KafkaClickTopic)
	server := httpapi.NewServer(cfg, st, rc, publisher)

	return &Application{
		cfg:       cfg,
		store:     st,
		cache:     rc,
		publisher: publisher,
		server:    server,
		shutdown:  shutdownTelemetry,
	}, nil
}

func (a *Application) Run() error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("urlshorter api listening on %s", a.cfg.HTTPAddr)
		errCh <- a.server.Listen()
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-signalCh:
		log.Printf("received signal %s, shutting down", sig)
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
	defer cancel()
	return errors.Join(
		a.server.Shutdown(ctx),
		a.publisher.Close(),
		a.cache.Close(),
		a.shutdown(ctx),
	)
}
