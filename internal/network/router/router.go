package router

import (
	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/network/handlers"
	"github.com/denmor86/ya-gophermart/internal/network/middleware"
	"github.com/denmor86/ya-gophermart/internal/services"
	"github.com/denmor86/ya-gophermart/internal/storage"
	"github.com/go-chi/chi/v5"

	"github.com/go-chi/jwtauth"
)

type Router struct {
	Config    config.Config
	Indentity *services.Identity
}

func NewRouter(config config.Config, storage *storage.Database) *Router {
	return &Router{
		Config:    config,
		Indentity: services.NewIdentity(config, storage),
	}
}

func (router *Router) HandleRouter() chi.Router {
	JWTAuth := router.Indentity.GetTokenAuth()
	//compressMiddleware := middleware.Compress(5, "gzip", "deflate")
	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.LogHandle)
		r.Route("/user", func(r chi.Router) {
			r.Post("/register", handlers.RegisterUserHandler(router.Indentity))
			r.Post("/login", handlers.AuthenticateUserHandle(router.Indentity))
			r.Group(func(r chi.Router) {
				r.Use(jwtauth.Verifier(JWTAuth))
				//r.Use(jwtauth.Authenticator(JWTAuth))
			})
		})
	})
	return r
}
