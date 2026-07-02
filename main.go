package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/thumbrise/xdebug-web/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	root := cmd.NewRootCMD(cmd.RootCMDLDFlags{
		Version: version,
		Commit:  commit,
		Date:    date,
	})

	if err := root.Run(ctx, os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fatal error: %[1]v\n", err)

		cancel()
		os.Exit(1)
	}

	cancel()
}
