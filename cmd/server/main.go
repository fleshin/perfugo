package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"perfugo/internal/config"
	"perfugo/internal/db"
	applog "perfugo/internal/log"
	"perfugo/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		applog.Error(context.Background(), "failed to load configuration", "error", err)
		os.Exit(1)
	}

	if err := applog.SetLevel(cfg.Logging.Level); err != nil {
		applog.Error(context.Background(), "invalid log level configuration", "level", cfg.Logging.Level, "error", err)
		os.Exit(1)
	}

	database, err := db.Configure(cfg.Database)
	if err != nil {
		applog.Error(context.Background(), "failed to initialize database", "error", err)
		os.Exit(1)
	}

	srv, err := server.New(server.Config{
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
		applog.Error(context.Background(), "failed to initialize http server", "error", err)
		os.Exit(1)
	}

	go func() {
		applog.Info(context.Background(), "starting http server", "addr", cfg.Server.Addr)
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			applog.Error(context.Background(), "server encountered an error", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	applog.Info(context.Background(), "shutting down http server")
	if err := srv.Stop(); err != nil {
		applog.Error(context.Background(), "graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}
