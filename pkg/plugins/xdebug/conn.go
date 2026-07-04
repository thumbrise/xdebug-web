package xdebug

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
)

const maxMessageSize = 10 << 20

var (
	errEmptyLength   = errors.New("empty length prefix")
	errSizeExceeded  = errors.New("message size exceeds limit")
	errNullTerm      = errors.New("expected null terminator")
	errLengthTooLong = errors.New("length prefix too long")
)

type conn struct {
	net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
}

func NewConn(raw net.Conn) *conn {
	return &conn{
		Conn:   raw,
		reader: bufio.NewReaderSize(raw, 64<<10),
	}
}

func (c *conn) ReadMessage(ctx context.Context) ([]byte, error) {
	lengthStr, err := c.readUntilNull(ctx)
	if err != nil {
		return nil, fmt.Errorf("read length prefix: %w", err)
	}

	if lengthStr == "" {
		return nil, errEmptyLength
	}

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid length %q: %w", lengthStr, err)
	}

	if length < 0 || length > maxMessageSize {
		return nil, fmt.Errorf("size %d: %w", length, errSizeExceeded)
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		return nil, fmt.Errorf("read payload (%d bytes): %w", length, err)
	}

	nb, err := c.reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("read trailing null: %w", err)
	}

	if nb != 0 {
		return nil, fmt.Errorf("got 0x%x: %w", nb, errNullTerm)
	}

	return payload, nil
}

func (c *conn) SendMessage(ctx context.Context, msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	deadline, ok := ctx.Deadline()
	if ok {
		if err := c.SetWriteDeadline(deadline); err != nil {
			return err
		}
	}

	_, err := c.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func (c *conn) readUntilNull(ctx context.Context) (string, error) {
	var buf []byte

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		b, err := c.reader.ReadByte()
		if err != nil {
			return "", err
		}

		if b == 0 {
			return string(buf), nil
		}

		buf = append(buf, b)

		if len(buf) > 32 {
			return "", fmt.Errorf("%q...: %w", string(buf), errLengthTooLong)
		}
	}
}
