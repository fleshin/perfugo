package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"gorm.io/gorm"

	"perfugo/internal/config"
	"perfugo/internal/db"
	"perfugo/internal/db/mock"
	applog "perfugo/internal/log"
	"perfugo/internal/server"
)

var (
	exitFunc             = os.Exit
	loadConfigFunc       = config.Load
	setLogLevelFunc      = applog.SetLevel
	configureDatabase    = db.Configure
	newMockDatabaseFunc  = mock.New
	newServerFunc        = func(cfg server.Config) (serverLifecycle, error) { return server.New(cfg) }
	subscribeShutdownSig = func() (<-chan os.Signal, func()) {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		return sigCh, func() { signal.Stop(sigCh) }
	}
)

type serverLifecycle interface {
	Start() error
	Stop() error
}

func main() {
	exitFunc(run(context.Background()))
}

func run(ctx context.Context) int {
	cfg, err := loadConfigFunc()
	if err != nil {
		applog.Error(ctx, "failed to load configuration", "error", err)
		return 1
	}

	applog.Debug(ctx, "configuration loaded", "addr", cfg.Server.Addr, "logLevel", cfg.Logging.Level)

	if err := setLogLevelFunc(cfg.Logging.Level); err != nil {
		applog.Error(ctx, "invalid log level configuration", "level", cfg.Logging.Level, "error", err)
		return 1
	}

	applog.Debug(ctx, "log level configured", "level", cfg.Logging.Level)

	var database *gorm.DB
	if cfg.Database.UseMock || strings.TrimSpace(cfg.Database.URL) == "" {
		applog.Info(ctx, "using in-memory mock database")
		database, err = newMockDatabaseFunc(ctx)
	} else {
		database, err = configureDatabase(cfg.Database)
	}
	if err != nil {
		applog.Error(ctx, "failed to initialize database", "error", err)
		return 1
	}

	applog.Debug(ctx, "database configured", "hasDB", database != nil)

	srv, err := newServerFunc(server.Config{
		Addr: cfg.Server.Addr,
		Session: server.SessionConfig{
			Lifetime:     cfg.Auth.Session.Lifetime,
			CookieName:   cfg.Auth.Session.CookieName,
			CookieDomain: cfg.Auth.Session.CookieDomain,
			CookieSecure: cfg.Auth.Session.CookieSecure,
		},
		Database: database,
	})
	if err != nil {
		applog.Error(ctx, "failed to initialize http server", "error", err)
		return 1
	}

	applog.Debug(ctx, "http server initialized", "addr", cfg.Server.Addr)

	serverErrCh := make(chan error, 1)
	go func() {
		applog.Info(ctx, "starting http server", "addr", cfg.Server.Addr)
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	shutdownCh, cancel := subscribeShutdownSig()
	defer cancel()

	select {
	case err := <-serverErrCh:
		if err != nil {
			applog.Error(ctx, "server encountered an error", "error", err)
			return 1
		}
		return 0
	case <-shutdownCh:
		applog.Info(ctx, "shutting down http server")
		if err := srv.Stop(); err != nil {
			applog.Error(ctx, "graceful shutdown failed", "error", err)
			return 1
		}
	}

	return 0
}
