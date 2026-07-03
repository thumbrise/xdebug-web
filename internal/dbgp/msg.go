package dbgp

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

const MaxMessageSize = 10 << 20

var (
	ErrEmptyLength   = errors.New("empty length prefix")
	ErrSizeExceeded  = errors.New("message size exceeds limit")
	ErrNullTerm      = errors.New("expected null terminator")
	ErrLengthTooLong = errors.New("length prefix too long")
)

type Conn struct {
	net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{
		Conn:   conn,
		reader: bufio.NewReaderSize(conn, 64<<10),
	}
}

func (c *Conn) ReadMessage(ctx context.Context) ([]byte, error) {
	lengthStr, err := c.readUntilNull(ctx)
	if err != nil {
		return nil, fmt.Errorf("read length prefix: %w", err)
	}

	if lengthStr == "" {
		return nil, ErrEmptyLength
	}

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid length %q: %w", lengthStr, err)
	}

	if length < 0 || length > MaxMessageSize {
		return nil, fmt.Errorf("size %d: %w", length, ErrSizeExceeded)
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
		return nil, fmt.Errorf("got 0x%x: %w", nb, ErrNullTerm)
	}

	return payload, nil
}

func (c *Conn) SendMessage(ctx context.Context, msg string) error {
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

func (c *Conn) Close() error {
	return c.Conn.Close()
}

func (c *Conn) readUntilNull(ctx context.Context) (string, error) {
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
			return "", fmt.Errorf("%q...: %w", string(buf), ErrLengthTooLong)
		}
	}
}
