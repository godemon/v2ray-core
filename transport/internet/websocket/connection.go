package websocket

import (
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"v2ray.com/core/common/buf"
	"v2ray.com/core/common/errors"
)

var (
	_ buf.MultiBufferReader = (*connection)(nil)
	_ buf.MultiBufferWriter = (*connection)(nil)
)

// connection is a wrapper for net.Conn over WebSocket connection.
type connection struct {
	conn   *websocket.Conn
	reader io.Reader

	mergingReader buf.Reader
	mergingWriter buf.Writer
}

func newConnection(conn *websocket.Conn) *connection {
	return &connection{
		conn: conn,
	}
}

// Read implements net.Conn.Read()
func (c *connection) Read(b []byte) (int, error) {
	for {
		reader, err := c.getReader()
		if err != nil {
			return 0, err
		}

		nBytes, err := reader.Read(b)
		if errors.Cause(err) == io.EOF {
			c.reader = nil
			continue
		}
		return nBytes, err
	}
}

func (c *connection) ReadMultiBuffer() (buf.MultiBuffer, error) {
	if c.mergingReader == nil {
		c.mergingReader = buf.NewMergingReader(c)
	}
	return c.mergingReader.Read()
}

func (c *connection) getReader() (io.Reader, error) {
	if c.reader != nil {
		return c.reader, nil
	}

	_, reader, err := c.conn.NextReader()
	if err != nil {
		return nil, err
	}
	c.reader = reader
	return reader, nil
}

// Write implements io.Writer.
func (c *connection) Write(b []byte) (int, error) {
	if err := c.conn.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *connection) WriteMultiBuffer(mb buf.MultiBuffer) error {
	if c.mergingWriter == nil {
		c.mergingWriter = buf.NewMergingWriter(c)
	}
	return c.mergingWriter.Write(mb)
}

func (c *connection) Close() error {
	c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second*5))
	return c.conn.Close()
}

func (c *connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *connection) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *connection) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *connection) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
