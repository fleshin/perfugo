package server

import (
	"context"
	"net/http"

	"perfugo/internal/handlers"
	applog "perfugo/internal/log"
)

func newRouter() http.Handler {
	mux := http.NewServeMux()
	applog.Debug(context.Background(), "registering http routes")
	mux.HandleFunc("/healthz", handlers.Health)
	applog.Debug(context.Background(), "route registered", "path", "/healthz")
	mux.HandleFunc("/login", handlers.Login)
	applog.Debug(context.Background(), "route registered", "path", "/login")
	mux.HandleFunc("/signup", handlers.Signup)
	applog.Debug(context.Background(), "route registered", "path", "/signup")
	mux.HandleFunc("/logout", handlers.Logout)
	applog.Debug(context.Background(), "route registered", "path", "/logout")
	mux.Handle("/app", handlers.RequireAuthentication(http.HandlerFunc(handlers.Dashboard)))
	mux.Handle("/app/", handlers.RequireAuthentication(http.HandlerFunc(handlers.Dashboard)))
	mux.Handle("/app/preferences/update", handlers.RequireAuthentication(http.HandlerFunc(handlers.UpdatePreferences)))
	applog.Debug(context.Background(), "route registered", "path", "/app", "protected", true)
	mux.HandleFunc("/", handlers.Home)
	applog.Debug(context.Background(), "route registered", "path", "/")
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/static"))))
	applog.Debug(context.Background(), "route registered", "path", "/assets/", "static", true)
	return mux
}
