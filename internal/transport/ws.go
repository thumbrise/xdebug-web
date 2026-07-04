package transport

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/thumbrise/xdebug-web/internal/dbgp"
	"github.com/thumbrise/xdebug-web/internal/session"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	store  *session.Store
	logger *slog.Logger
	mu     sync.RWMutex
	conns  map[*websocket.Conn]context.CancelFunc
}

func NewHandler(store *session.Store, logger *slog.Logger) *Handler {
	return &Handler{
		store:  store,
		logger: logger.With("subsystem", "ws"),
		conns:  make(map[*websocket.Conn]context.CancelFunc),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.WarnContext(r.Context(), "upgrade", "error", err)

		return
	}

	ctx, cancel := context.WithCancel(r.Context())

	h.mu.Lock()
	h.conns[conn] = cancel
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.conns, conn)
		h.mu.Unlock()
		cancel()

		_ = conn.Close()
	}()

	sub := h.subscribeToSession(ctx)
	if sub != nil {
		defer h.store.Active().Unsubscribe(sub)
	}

	done := make(chan struct{})

	go h.readLoop(ctx, conn, done)

	h.writeLoop(ctx, conn, done, sub)
}

func (h *Handler) writeLoop(ctx context.Context, conn *websocket.Conn, done chan struct{}, sub chan *session.State) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-done:
			return

		case state, ok := <-sub:
			if !ok {
				return
			}

			msg := OutgoingMessage{
				Type: "state",
				Data: state,
			}

			data, err := json.Marshal(msg)
			if err != nil {
				h.logger.WarnContext(ctx, "marshal", "error", err)

				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				h.logger.WarnContext(ctx, "write", "error", err)

				return
			}
		}
	}
}

func (h *Handler) readLoop(_ context.Context, conn *websocket.Conn, done chan struct{}) {
	defer close(done)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var cmd IncomingCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			continue
		}

		active := h.store.Active()
		if active == nil {
			continue
		}

		dbgpCmd, err := toDBGpCommand(cmd)
		if err != nil {
			continue
		}

		active.SendCommand(dbgpCmd)
	}
}

func (h *Handler) subscribeToSession(_ context.Context) chan *session.State {
	active := h.store.Active()
	if active == nil {
		return nil
	}

	return active.Subscribe()
}

func toDBGpCommand(cmd IncomingCommand) (dbgp.Command, error) {
	var cmdType dbgp.CommandType

	switch cmd.Type {
	case "step_into":
		cmdType = dbgp.CmdStepInto
	case "step_over":
		cmdType = dbgp.CmdStepOver
	case "step_out":
		cmdType = dbgp.CmdStepOut
	case "run":
		cmdType = dbgp.CmdRun
	case "stop":
		cmdType = dbgp.CmdStop
	case "breakpoint_set":
		cmdType = dbgp.CmdBreakpointSet
	case "breakpoint_remove":
		cmdType = dbgp.CmdBreakpointRemove
	default:
		return dbgp.Command{}, dbgp.ErrUnknownCommand
	}

	return dbgp.Command{Type: cmdType, Args: cmd.Args}, nil
}
