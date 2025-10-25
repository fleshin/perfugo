package server

import (
	"net/http"

	"perfugo/internal/handlers"
)

func newRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handlers.Health)
	mux.HandleFunc("/login", handlers.Login)
	mux.HandleFunc("/app", handlers.Dashboard)
	mux.HandleFunc("/", handlers.Home)
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/static"))))
	return mux
}
