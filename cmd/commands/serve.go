package commands

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"
)

var (
	flagServePort    string
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
		&cli.StringFlag{
			Name:        "port",
			Usage:       "Port to serve web server on",
			Required:    false,
			Value:       "8000",
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
		fmt.Println("Hello World")

		return nil
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
