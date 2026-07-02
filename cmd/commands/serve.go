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
	flagServeProjectDir string
)

func projectDirOrDefault(dir string) string {
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
  xdebug-web serve --project-dir /var/www/html
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
			Name:        "project-dir",
			Usage:       "Path to the PHP project root",
			Required:    false,
			Destination: &flagServeProjectDir,
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
			ProjectDir: filepath.Clean(projectDirOrDefault(flagServeProjectDir)),
		}

		logger.InfoContext(ctx, "project directory", "path", cfg.ProjectDir)

		server := web.NewServer(cfg, logger)

		return server.Serve(ctx)
	},
}
