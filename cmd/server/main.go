package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"perfugo/internal/server"
)

func main() {
	cfg := server.Config{
		Addr: getEnv("ADDR", ":8080"),
	}

	srv := server.New(cfg)

	go func() {
		log.Printf("starting http server on %s", cfg.Addr)
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

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
