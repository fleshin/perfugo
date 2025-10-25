package server

import (
	"net/http"

	"perfugo/internal/handlers"
)

func newRouter(providers []handlers.OIDCProvider) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handlers.Health)
	mux.HandleFunc("/login", handlers.Login)
	mux.HandleFunc("/logout", handlers.Logout)
	for _, provider := range providers {
		mux.HandleFunc("/auth/oidc/"+provider.ID+"/start", handlers.OIDCStartHandler(provider.ID))
		mux.HandleFunc("/auth/oidc/"+provider.ID+"/callback", handlers.OIDCCallbackHandler(provider.ID))
	}
	mux.Handle("/app", handlers.RequireAuthentication(http.HandlerFunc(handlers.Dashboard)))
	mux.HandleFunc("/", handlers.Home)
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/static"))))
	return mux
}
