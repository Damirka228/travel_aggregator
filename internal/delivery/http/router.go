package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Router struct {
	travelHandler  *Handler
	authHandler    *AuthHandler
	enableProfiler bool
}

func NewRouter(travelHandler *Handler, authHandler *AuthHandler, enableProfiler bool) *Router {
	return &Router{
		travelHandler:  travelHandler,
		authHandler:    authHandler,
		enableProfiler: enableProfiler,
	}
}

func (r *Router) SetupRoutes() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/destinations", r.travelHandler.Destinations)
	router.Post("/auth/signup", r.authHandler.SignUp)
	router.Post("/auth/signin", r.authHandler.SignIn)

	if r.enableProfiler {
		router.Mount("/debug", middleware.Profiler())
	}

	return router
}
