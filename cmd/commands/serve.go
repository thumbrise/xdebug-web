package commands

import (
	"context"
	"os"
	"path/filepath"

	"github.com/thumbrise/xdebug-web/internal/infra"
	"github.com/thumbrise/xdebug-web/internal/web"
	"github.com/urfave/cli/v3"
)

var (
	flagServePort       int
	flagServeVerbose    bool
	flagServeDev        bool
	flagServeRoot       string
	flagServeRootRemote string
	flagServeDbgpPort   int
	flagServeDbgpAddr   string
)

func rootOrDefault(dir string) string {
	if dir != "" {
		return dir
	}

	cwd, _ := os.Getwd()

	return cwd
}

var ServeCMD = &cli.Command{
	Name:      "serve",
	Usage:     "",
	UsageText: "xdebug-web serve --port <value>",
	Description: `Run web server.

Examples:
  xdebug-web serve --port 8080
  xdebug-web serve --dev --port 8080
  xdebug-web serve --root /var/www/html
  xdebug-web serve --root /Users/user/project --root-remote /var/www/html
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
		&cli.BoolFlag{
			Name:        "dev",
			Usage:       "Development mode (no embedded frontend, CORS enabled)",
			Required:    false,
			Destination: &flagServeDev,
		},
		&cli.StringFlag{
			Name:        "root",
			Usage:       "Path to the PHP project root on the host filesystem",
			Required:    false,
			Destination: &flagServeRoot,
		},
		&cli.StringFlag{
			Name:        "root-remote",
			Usage:       "Remote root path in Xdebug file:// URIs (e.g. /var/www/html). Empty if not using Docker path mapping",
			Required:    false,
			Destination: &flagServeRootRemote,
		},
		&cli.IntFlag{
			Name:        "dbgp-port",
			Usage:       "Port for the DBGp (Xdebug) listener",
			Required:    false,
			Value:       9003,
			Destination: &flagServeDbgpPort,
		},
		&cli.StringFlag{
			Name:        "dbgp-addr",
			Usage:       "Address for the DBGp (Xdebug) listener (overrides --dbgp-port)",
			Required:    false,
			Destination: &flagServeDbgpAddr,
		},
	},

	Action: func(ctx context.Context, cmd *cli.Command) error {
		logger := infra.NewLogger(os.Stderr, flagServeVerbose)

		cfg := web.Config{
			AppName:    "Xdebug-web",
			Port:       flagServePort,
			Version:    "1.0.0",
			DocsPath:   "/api/docs",
			DevMode:    flagServeDev,
			Root:       filepath.Clean(rootOrDefault(flagServeRoot)),
			RootRemote: flagServeRootRemote,
			DbgpPort:   flagServeDbgpPort,
			DbgpAddr:   flagServeDbgpAddr,
		}

		logger.InfoContext(ctx, "root directory", "path", cfg.Root, "remoteRoot", cfg.RootRemote)

		server := web.NewServer(cfg, logger)

		return server.Serve(ctx)
	},
}
