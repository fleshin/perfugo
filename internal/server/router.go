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
	mux.Handle("/app/sections/ingredients/new", handlers.RequireAuthentication(http.HandlerFunc(handlers.IngredientNew)))
	mux.Handle("/app/sections/ingredients/create", handlers.RequireAuthentication(http.HandlerFunc(handlers.IngredientCreate)))
	mux.Handle("/app/sections/ingredients/delete", handlers.RequireAuthentication(http.HandlerFunc(handlers.IngredientDelete)))
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/ingredients/table", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/ingredients/detail", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/ingredients/edit", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/ingredients/update", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/ingredients/new", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/ingredients/create", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/ingredients/delete", "protected", true)
	mux.Handle("/app/sections/formulas/list", handlers.RequireAuthentication(http.HandlerFunc(handlers.FormulaList)))
	mux.Handle("/app/sections/formulas/detail", handlers.RequireAuthentication(http.HandlerFunc(handlers.FormulaDetail)))
	mux.Handle("/app/sections/formulas/create", handlers.RequireAuthentication(http.HandlerFunc(handlers.FormulaCreate)))
	mux.Handle("/app/sections/formulas/edit", handlers.RequireAuthentication(http.HandlerFunc(handlers.FormulaEdit)))
	mux.Handle("/app/sections/formulas/update", handlers.RequireAuthentication(http.HandlerFunc(handlers.FormulaUpdate)))
	mux.Handle("/app/sections/formulas/ingredient-row", handlers.RequireAuthentication(http.HandlerFunc(handlers.FormulaIngredientRow)))
	mux.Handle("/app/sections/formulas/delete", handlers.RequireAuthentication(http.HandlerFunc(handlers.FormulaDelete)))
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/formulas/list", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/formulas/detail", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/formulas/create", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/formulas/edit", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/formulas/update", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/formulas/ingredient-row", "protected", true)
	applog.Debug(context.Background(), "route registered", "path", "/app/sections/formulas/delete", "protected", true)
	mux.HandleFunc("/", handlers.Home)
	applog.Debug(context.Background(), "route registered", "path", "/")
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/static"))))
	applog.Debug(context.Background(), "route registered", "path", "/assets/", "static", true)
	return mux
}
