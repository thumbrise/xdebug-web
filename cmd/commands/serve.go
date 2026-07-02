package commands

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/thumbrise/xdebug-web/internal"
	"github.com/urfave/cli/v3"
)

var (
	flagServePort    int
	flagServeVerbose bool
)

var ServeCMD = &cli.Command{
	Name:      "serve",
	Usage:     "",
	UsageText: "xdebug-web serve --port <value>",
	Description: `Run web server.

Examples:
  xdebug-web serve --port 8000
`,
	Suggest: true,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "port",
			Usage:       "Port to serve web server on",
			Required:    false,
			Value:       8080,
			Destination: &flagServePort,
		},
		&cli.BoolFlag{
			Name:        "verbose",
			Aliases:     []string{"v"},
			Usage:       "Verbose output",
			Required:    false,
			Destination: &flagServeVerbose,
		},
	},

	Action: func(ctx context.Context, cmd *cli.Command) error {
		configureLogger(os.Stderr, flagServeVerbose)

		server := internal.NewServer(slog.Default(), flagServePort)

		return server.Serve(ctx)
	},
}

func configureLogger(writer io.Writer, verbose bool) {
	opts := &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelInfo,
	}

	if verbose {
		opts.Level = slog.LevelDebug
		opts.AddSource = true
	}

	handler := slog.NewTextHandler(writer, opts)

	logger := slog.New(handler)

	slog.SetDefault(logger)
}
