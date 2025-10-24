package server

import (
	"net/http"

	"perfugo/internal/handlers"
)

func newRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handlers.Health)
	mux.Handle("/", http.FileServer(http.Dir("web/static")))
	return mux
}
