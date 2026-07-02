package internal

import (
	"context"
	"log/slog"
)

type Server struct {
	port   int
	logger *slog.Logger
}

func NewServer(logger *slog.Logger, port int) *Server {
	return &Server{logger: logger, port: port}
}

func (s *Server) Serve(ctx context.Context) error {
	if s.port <= 0 {
		s.port = 8080
		s.logger.WarnContext(ctx, "server port is invalid, set to default")
	}

	s.logger.InfoContext(ctx, "server started", "port", s.port)

	return nil
}
