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

const maxMsgBuf = 64

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

	stateCh := h.subscribeToSessions(ctx)

	done := make(chan struct{})

	go h.readLoop(ctx, conn, done)

	h.writeLoop(ctx, conn, done, stateCh)
}

func (h *Handler) writeLoop(ctx context.Context, conn *websocket.Conn, done chan struct{}, stateCh chan OutgoingMessage) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-done:
			return

		case msg, ok := <-stateCh:
			if !ok {
				return
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

		pluginCmd, err := toPluginCommand(cmd)
		if err != nil {
			continue
		}

		if cmd.Type == "breakpoint_set" || cmd.Type == "breakpoint_remove" {
			if cmd.Type == "breakpoint_set" {
				h.store.SaveBreakpoint(cmd.Args["file"], cmd.Args["line"])
			} else {
				h.store.DeleteBreakpoint(cmd.Args["file"], cmd.Args["line"])
			}

			for _, sess := range h.store.All() {
				sess.SendCommand(pluginCmd)
			}
		} else {
			sess := h.store.Active()
			if sess == nil {
				continue
			}

			sess.SendCommand(pluginCmd)
		}
	}
}

func (h *Handler) subscribeToSessions(ctx context.Context) chan OutgoingMessage {
	ch := make(chan OutgoingMessage, maxMsgBuf)

	for _, sess := range h.store.All() {
		h.forwardSession(ctx, sess, ch)
	}

	newSessions := h.store.SubscribeNew()

	go func() {
		<-ctx.Done()
		h.store.UnsubscribeNew(newSessions)
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case sess := <-newSessions:
				h.forwardSession(ctx, sess, ch)
			}
		}
	}()

	return ch
}

func (h *Handler) forwardSession(ctx context.Context, sess *session.Session, ch chan OutgoingMessage) {
	sub := sess.Subscribe()

	go func() {
		defer sess.Unsubscribe(sub)

		for {
			select {
			case <-ctx.Done():
				return

			case state, ok := <-sub:
				if !ok {
					return
				}

				msg := OutgoingMessage{
					Type:   "state",
					Data:   state,
					Source: sess.ID(),
				}

				select {
				case ch <- msg:
				default:
				}
			}
		}
	}()
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
