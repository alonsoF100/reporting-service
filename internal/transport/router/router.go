package router

import (
	"github.com/alonsoF100/reporting-service/internal/transport/handler"
	"github.com/go-chi/chi/v5"
)

type Router struct {
	Handler *handler.Handler
}

func New(handler *handler.Handler) *Router {
	return &Router{
		Handler: handler,
	}
}

func (rt Router) Setup() *chi.Mux {
	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/devices/{id}", rt.Handler.GetDeviceMessages)
	})

	return r
}
