package server

import (
	"log/slog"
	"net/http"

	"github.com/alonsoF100/reporting-service/internal/config"
	"github.com/alonsoF100/reporting-service/internal/transport/handler"
	"github.com/alonsoF100/reporting-service/internal/transport/router"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	Server *http.Server
	Router *chi.Mux
	Cfg    *config.Config
	Logger *slog.Logger
}

func New(cfg *config.Config, handlers *handler.Handler, logger *slog.Logger) *Server {
	rtr := router.New(handlers).Setup()

	stdLogger := slog.NewLogLogger(logger.Handler(), slog.LevelError)

	srv := &http.Server{
		Addr:         cfg.Server.PortStr(),
		Handler:      rtr,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		ErrorLog:     stdLogger,
	}

	return &Server{
		Server: srv,
		Router: rtr,
		Cfg:    cfg,
		Logger: logger,
	}
}

func (s *Server) Start() error {
	slog.Info("Starting HTTP server",
		slog.String("port", s.Cfg.Server.PortStr()),
	)

	return s.Server.ListenAndServe()
}
