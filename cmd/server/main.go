package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"perfugo/internal/config"
	"perfugo/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	srv := server.New(server.Config{Addr: cfg.Server.Addr})

	go func() {
		log.Printf("starting http server on %s", cfg.Server.Addr)
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server encountered an error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	log.Println("shutting down http server")
	if err := srv.Stop(); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
}
