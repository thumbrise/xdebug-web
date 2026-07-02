package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	logger *slog.Logger
	config Config
}

func NewServer(cfg Config, logger *slog.Logger) *Server {
	return &Server{logger: logger, config: cfg}
}

func (s *Server) Serve(ctx context.Context) error {
	if s.config.Port <= 0 {
		s.config.Port = 8080
		s.logger.WarnContext(ctx, "server port is invalid, set to default")
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("welcome"))
	})

	humacfg := huma.DefaultConfig(s.config.AppName, s.config.Version)
	humacfg.DocsPath = s.config.DocsPath
	_ = humachi.New(r, humacfg)

	srv := http.Server{
		Addr:              ":" + strconv.Itoa(s.config.Port),
		Handler:           r,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		slog.Info("shutting down server")

		shutdownCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			10*time.Second,
		)
		defer cancel()

		err := srv.Shutdown(shutdownCtx)

		if errors.Is(err, context.DeadlineExceeded) {
			slog.WarnContext(shutdownCtx, "shutdown timed out, some connections were force-closed")
		} else if err != nil {
			slog.ErrorContext(shutdownCtx, "shutdown error", "error", err)
		}
	}()

	s.logger.InfoContext(ctx, "server started", "port", s.config.Port)

	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
