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
	mux.Handle("/app/sections/ingredients/table", handlers.RequireAuthentication(http.HandlerFunc(handlers.IngredientTable)))
	mux.Handle("/app/sections/ingredients/detail", handlers.RequireAuthentication(http.HandlerFunc(handlers.IngredientDetail)))
	mux.Handle("/app/sections/ingredients/edit", handlers.RequireAuthentication(http.HandlerFunc(handlers.IngredientEdit)))
	mux.Handle("/app/sections/ingredients/update", handlers.RequireAuthentication(http.HandlerFunc(handlers.IngredientUpdate)))
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/ingredients/table", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/ingredients/detail", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/ingredients/edit", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/ingredients/update", "protected", true)
	mux.Handle("/app/sections/formulas/list", handlers.RequireAuthentication(http.HandlerFunc(handlers.FormulaList)))
	mux.Handle("/app/sections/formulas/detail", handlers.RequireAuthentication(http.HandlerFunc(handlers.FormulaDetail)))
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/formulas/list", "protected", true)
	mux.HandleFunc("/", handlers.Home)
	applog.Debug(context.Background(), "route registered", "path", "/")
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/static"))))
	applog.Debug(context.Background(), "route registered", "path", "/assets/", "static", true)
	return mux
}
