package transport

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/thumbrise/xdebug-web/internal/session"
	"github.com/thumbrise/xdebug-web/pkg/plugins"
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

func (h *Handler) writeLoop(ctx context.Context, conn *websocket.Conn, done chan struct{}, sub chan *plugins.State) {
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
				Type:   "state",
				Data:   state,
				Source: h.store.Active().ID(),
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

		sess := h.store.Active()
		if sess == nil {
			continue
		}

		pluginCmd, err := toPluginCommand(cmd)
		if err != nil {
			continue
		}

		sess.SendCommand(pluginCmd)
	}
}

func (h *Handler) subscribeToSession(_ context.Context) chan *plugins.State {
	active := h.store.Active()
	if active == nil {
		return nil
	}

	return active.Subscribe()
}

func toPluginCommand(cmd IncomingCommand) (plugins.Command, error) {
	var cmdType plugins.CommandType

	switch cmd.Type {
	case "step_into":
		cmdType = plugins.CmdStepInto
	case "step_over":
		cmdType = plugins.CmdStepOver
	case "step_out":
		cmdType = plugins.CmdStepOut
	case "run":
		cmdType = plugins.CmdRun
	case "stop":
		cmdType = plugins.CmdStop
	case "breakpoint_set":
		cmdType = plugins.CmdBreakpointSet
	case "breakpoint_remove":
		cmdType = plugins.CmdBreakpointRemove
	default:
		return plugins.Command{}, plugins.ErrUnknownCommand
	}

	return plugins.Command{Type: cmdType, Args: cmd.Args}, nil
}
