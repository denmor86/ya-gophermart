package router

import (
	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/network/handlers"
	"github.com/denmor86/ya-gophermart/internal/network/middleware"
	"github.com/denmor86/ya-gophermart/internal/services"
	"github.com/denmor86/ya-gophermart/internal/storage"
	"github.com/go-chi/chi/v5"
	chi_middleware "github.com/go-chi/chi/v5/middleware"

	"github.com/go-chi/jwtauth/v5"
)

type Router struct {
	Config    config.Config
	Indentity services.IdentityService
	Orders    services.OrdersService
	Loyalty   services.LoyaltyService
}

func NewRouter(config config.Config, storage storage.Storage) *Router {
	return &Router{
		Config:    config,
		Indentity: services.NewIdentity(config.Server.JWTSecret, storage.Users),
		Orders:    services.NewOrders(services.NewAccrualService(config.Accrual.AccrualAddr), storage.Orders, storage.Users),
		Loyalty:   services.NewLoyalty(storage.Loyaltys, storage.Users),
	}
}

func (router *Router) HandleRouter() chi.Router {
	ja := router.Indentity.GetTokenAuth()
	compressMiddleware := chi_middleware.Compress(5, "gzip", "deflate")
	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.LogHandle)
		r.Route("/user", func(r chi.Router) {
			r.Post("/register", handlers.RegisterUserHandler(router.Indentity))
			r.Post("/login", handlers.AuthenticateUserHandle(router.Indentity))
			r.Group(func(r chi.Router) {
				r.Use(jwtauth.Verifier(ja))
				r.Use(jwtauth.Authenticator(ja))
				r.Post("/orders", handlers.OrdersHandler(router.Orders))
				r.With(compressMiddleware).Get("/orders", handlers.GetOrdersHandler(router.Orders))
				r.Route("/balance", func(r chi.Router) {
					r.With(compressMiddleware).Get("/", handlers.GetUserBalanceHandler(router.Loyalty))
					r.Post("/withdraw", handlers.WithdrawHandler(router.Loyalty))
				})
				r.With(compressMiddleware).Get("/withdrawals", handlers.GetWithdrawHandler(router.Loyalty))
			})
		})
	})
	return r
}
