package web

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/thumbrise/xdebug-web/internal/web/handler"
)

type Server struct {
	logger *slog.Logger
	config Config
}

func NewServer(cfg Config, logger *slog.Logger) *Server {
	return &Server{logger: logger, config: cfg}
}

//nolint:funlen
func (s *Server) Serve(ctx context.Context) error {
	if s.config.Port <= 0 {
		s.config.Port = 8080
		s.logger.WarnContext(ctx, "server port is invalid, set to default")
	}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", handler.Health())
		r.Get("/files", handler.Files(s.config.ProjectDir))
		r.Get("/file", handler.File(s.config.ProjectDir))
	})

	if !s.config.DevMode {
		distFS := DistFS()

		r.Get("/", spaIndex(distFS))
		r.Get("/index.html", spaIndex(distFS))
		r.Handle("/assets/*", spaStatic(distFS))
		r.NotFound(spaFallback(distFS))
	}

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
		s.logger.Info("shutting down server")

		shutdownCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			10*time.Second,
		)
		defer cancel()

		err := srv.Shutdown(shutdownCtx)

		if errors.Is(err, context.DeadlineExceeded) {
			s.logger.WarnContext(shutdownCtx, "shutdown timed out, some connections were force-closed")
		} else if err != nil {
			s.logger.ErrorContext(shutdownCtx, "shutdown error", "error", err)
		}
	}()

	s.logger.InfoContext(ctx, "server started",
		"port", s.config.Port,
		"dev", s.config.DevMode,
	)

	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func spaIndex(distFS fs.FS) http.HandlerFunc {
	data, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		panic("xdebug-web: failed to read embedded index.html: " + err.Error())
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	}
}

func spaStatic(distFS fs.FS) http.HandlerFunc {
	return http.FileServer(http.FS(distFS)).ServeHTTP
}

func spaFallback(distFS fs.FS) http.HandlerFunc {
	return spaIndex(distFS)
}
