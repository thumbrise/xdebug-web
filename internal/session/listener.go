package session

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	xdebug "github.com/thumbrise/xdebug-web/pkg/plugins/xdebug"
)

type Listener struct {
	addr       string
	store      *Store
	logger     *slog.Logger
	mu         sync.Mutex
	running    bool
	remoteRoot string
	root       string
}

func NewListener(addr string, store *Store, logger *slog.Logger, remoteRoot, root string) *Listener {
	return &Listener{
		addr:       addr,
		store:      store,
		logger:     logger.With("subsystem", "listener"),
		remoteRoot: remoteRoot,
		root:       root,
	}
}

func (l *Listener) Addr() string {
	return l.addr
}

func (l *Listener) Listen(ctx context.Context) error {
	lc := net.ListenConfig{}

	listener, err := lc.Listen(ctx, "tcp", l.addr)
	if err != nil {
		return fmt.Errorf("listen tcp %s: %w", l.addr, err)
	}

	l.mu.Lock()
	l.running = true
	l.mu.Unlock()

	l.logger.InfoContext(ctx, "listening for xdebug connections", "addr", l.addr)

	go func() {
		<-ctx.Done()

		_ = listener.Close()
	}()

	var sessID int

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			l.logger.WarnContext(ctx, "accept", "error", err)

			continue
		}

		l.logger.InfoContext(ctx, "xdebug connected", "remote", conn.RemoteAddr())

		sessID++
		dbg := xdebug.NewDebugger(xdebug.NewConn(conn), l.logger, l.root, l.remoteRoot)
		sess := NewSession(fmt.Sprintf("xdebug-%d", sessID), dbg)

		l.store.Add(sess)

		go func() {
			defer func() {
				l.store.Remove(sess)
				dbg.Close()
				l.logger.InfoContext(ctx, "xdebug disconnected", "remote", conn.RemoteAddr())
			}()

			if err := sess.Run(ctx); err != nil && ctx.Err() == nil {
				l.logger.WarnContext(ctx, "session", "error", err)
			}
		}()
	}
}
