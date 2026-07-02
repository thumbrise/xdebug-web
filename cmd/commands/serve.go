package commands

import (
	"context"
	"os"

	"github.com/thumbrise/xdebug-web/internal/infra"
	"github.com/thumbrise/xdebug-web/internal/web"
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
  xdebug-web serve --port 8080
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
		logger := infra.NewLogger(os.Stderr, flagServeVerbose)

		cfg := web.Config{
			AppName:  "Xdebug-web",
			Port:     flagServePort,
			Version:  "1.0.0",
			DocsPath: "/api/docs",
		}
		server := web.NewServer(cfg, logger)

		return server.Serve(ctx)
	},
}
