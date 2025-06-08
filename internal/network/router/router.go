package router

import (
	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/network/handlers"
	"github.com/denmor86/ya-gophermart/internal/network/middleware"
	"github.com/denmor86/ya-gophermart/internal/services"
	"github.com/denmor86/ya-gophermart/internal/storage"
	"github.com/go-chi/chi/v5"

	"github.com/go-chi/jwtauth/v5"
)

type Router struct {
	Config    config.Config
	Indentity services.IdentityService
	Orders    services.OrdersService
}

func NewRouter(config config.Config, storage storage.IStorage) *Router {
	return &Router{
		Config:    config,
		Indentity: services.NewIdentity(config, storage),
		Orders:    services.NewOrders(storage),
	}
}

func (router *Router) HandleRouter() chi.Router {
	ja := router.Indentity.GetTokenAuth()
	//compressMiddleware := middleware.Compress(5, "gzip", "deflate")
	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.LogHandle)
		r.Route("/user", func(r chi.Router) {
			r.Post("/register", handlers.RegisterUserHandler(router.Indentity))
			r.Post("/login", handlers.AuthenticateUserHandle(router.Indentity))
			r.Group(func(r chi.Router) {
				r.Use(jwtauth.Verifier(ja))
				r.Use(jwtauth.Authenticator(ja))
				r.Post("/orders", handlers.NewOrdersHandler(router.Orders))
			})
		})
	})
	return r
}
