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
	mux.Handle("/app/preferences", handlers.RequireAuthentication(http.HandlerFunc(handlers.Preferences)))
	applog.Debug(context.Background(), "route registered", "path", "/app/preferences", "protected", true)
	mux.Handle("/app", handlers.RequireAuthentication(http.HandlerFunc(handlers.Dashboard)))
	mux.Handle("/app/", handlers.RequireAuthentication(http.HandlerFunc(handlers.Dashboard)))
	applog.Debug(context.Background(), "route registered", "path", "/app", "protected", true)
	mux.Handle("/app/api/aroma-chemicals", handlers.RequireAuthentication(http.HandlerFunc(handlers.AromaChemicalResource)))
	mux.Handle("/app/api/aroma-chemicals/", handlers.RequireAuthentication(http.HandlerFunc(handlers.AromaChemicalResource)))
	applog.Debug(context.Background(), "route registered", "path", "/app/api/aroma-chemicals", "protected", true)
	mux.Handle("/app/api/formula-ingredients", handlers.RequireAuthentication(http.HandlerFunc(handlers.FormulaIngredientResource)))
	mux.Handle("/app/api/formula-ingredients/", handlers.RequireAuthentication(http.HandlerFunc(handlers.FormulaIngredientResource)))
	applog.Debug(context.Background(), "route registered", "path", "/app/api/formula-ingredients", "protected", true)
	mux.HandleFunc("/", handlers.Home)
	applog.Debug(context.Background(), "route registered", "path", "/")
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/static"))))
	applog.Debug(context.Background(), "route registered", "path", "/assets/", "static", true)
	return mux
}
