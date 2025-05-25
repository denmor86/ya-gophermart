package router

import (
	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/network/middleware"
	"github.com/go-chi/chi/v5"
)

type Router struct {
	Config config.Config
}

func NewRouter(config config.Config) *Router {
	return &Router{Config: config}
}

func (router *Router) HandleRouter() chi.Router {
	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.LogHandle)
		r.Route("/user", func(r chi.Router) {
		})
	})
	return r
}
