package websocket

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"sync"
	"time"
)

// Conn represents a WebSocket connection. All public methods on Conn
// are safe to be simultaneously called.
type Conn struct {
	underlying io.ReadWriteCloser
	rmx        sync.Mutex
	wmx        sync.Mutex
	closed     bool

	pingCtx    context.Context
	pingCancel context.CancelFunc
	pingMx     sync.Mutex
}

// From returns a new WebSocket Conn from a value with a type that
// implements the io.ReadWriteCloser interface, notably net.Conn.
// It is expected that this connection will not be read from,
// written to, or closed once passed into this function.
func From(c io.ReadWriteCloser) *Conn {
	return &Conn{underlying: c, rmx: sync.Mutex{}, wmx: sync.Mutex{}, closed: false}
}

// Close marks the connection as closed and closes the underlying
// connection. It may return an error if there is an issue closing
// the underlying connection.
func (c *Conn) Close() error {
	c.rmx.Lock()
	c.wmx.Lock()
	defer c.rmx.Unlock()
	defer c.wmx.Unlock()
	c.closed = true
	return c.underlying.Close()
}

// Read reads a WebSocket frame from the underlying connection. If there
// is an issue reading the frame or the frame is malformed, it may return
// an error.
func (c *Conn) Read() (*Message, error) {
	c.rmx.Lock()
	defer c.rmx.Unlock()
	if c.closed {
		return nil, ErrConnectionClosed
	}
	message := new(Message)

	header := make([]byte, 2) // includes fin, rsv1, rsv2, rsv3, and opcode

	n, err := c.underlying.Read(header)
	if err != nil {
		return nil, ErrConnectionRead
	}
	if n != 2 {
		return nil, ErrMalformedFrame
	}

	fin := (header[0] & 0x80) != 0

	// FIXME: fragmented frames are not supported
	if !fin { // if frame is fragmented (0 means fragmented, 1 means final)
		return nil, ErrMalformedFrame
	}

	rsv1 := (header[0] & 0x40) != 0
	rsv2 := (header[0] & 0x20) != 0
	rsv3 := (header[0] & 0x10) != 0

	if rsv1 || rsv2 || rsv3 { // for extensions
		return nil, ErrMalformedFrame
	}

	// op-coding
	opcode := header[0] & 0x0F
	switch opcode {
	case 0x0:
	case 0x1:
		message.Type = MessageText
	case 0x2:
		message.Type = MessageBinary
	case 0x8:
		c.Close()
		message.Type = MessageClose
	case 0x9:
		err := c.Write(&Message{
			Type: MessagePong,
			Data: []byte{},
		})
		if err != nil {
			slog.Error("an error occured while sending pong as response to a ping", "error", err.Error())
		}
		message.Type = MessagePing
	case 0xA:
		c.pingMx.Lock()
		if c.pingCancel != nil {
			c.pingCancel()
		}
		c.pingCtx = nil
		c.pingCancel = nil
		c.pingMx.Unlock()
		message.Type = MessagePong
	default:
		return nil, ErrMalformedFrame
	}

	// payload length
	payloadLength := int(header[1] & 0x7F) // extenstion data + application data in bytes
	switch payloadLength {
	case 126: // the following 16 bits (or 2 bytes) is the uint payload length
		extendedPayloadLen := make([]byte, 2)
		_, err = c.underlying.Read(extendedPayloadLen)
		if err != nil {
			return nil, ErrConnectionRead
		}
		payloadLength = int(binary.BigEndian.Uint16(extendedPayloadLen))
	case 127: // the following 64 bits (or 8 bytes) is the uint payload length
		extendedPayloadLen := make([]byte, 8)
		_, err = c.underlying.Read(extendedPayloadLen)
		if err != nil {
			return nil, ErrConnectionRead
		}
		payloadLength = int(binary.BigEndian.Uint64(extendedPayloadLen))
	}

	// mask key
	isMasked := ((header[1] >> 7) & 1) != 0
	var maskKey []byte
	if isMasked {
		maskKey = make([]byte, 4)
		_, err = c.underlying.Read(maskKey)
		if err != nil {
			return nil, ErrConnectionRead
		}
	}

	// the actual payload
	payload := make([]byte, payloadLength)
	_, err = c.underlying.Read(payload)
	if err != nil {
		return nil, ErrConnectionRead
	}

	// unmask with xor
	if isMasked {
		for i := range payloadLength {
			payload[i] ^= maskKey[i%4] // xor =
		}
	}

	message.Data = payload
	return message, nil
}

// Write takes in a message and writes it as a WebSocket frame
// to the underlying connection.
func (c *Conn) Write(message *Message) error {
	c.wmx.Lock()
	defer c.wmx.Unlock()
	messageType := message.Type
	data := message.Data

	// 10 for header max size, messageType (1), payloadLength (1), extendedPayloadLength(8, depends on payloadLength)
	frame := make([]byte, 0, 10+len(data))
	fin := byte(0x80)    // 1000 0000 (indicates final frame)
	switch messageType { // fin (always 1), rsv1, rsv2, rsv3 (always 0), opcode
	case MessageText: // 1000 0001 -> 0x81
		frame = append(frame, fin|0x1)
	case MessageBinary: // 1000 0010 -> 0x82
		frame = append(frame, fin|0x2)
	case MessageClose: // 1000 1000 -> 0x88
		frame = append(frame, fin|0x8)
	case MessagePing: // 1000 1001 -> 0x89
		frame = append(frame, fin|0x9)
	case MessagePong: // 1000 1010 -> 0x8A
		frame = append(frame, fin|0xA)
	}

	payloadLength := len(data)
	// mask key and payload length
	if payloadLength < 126 { // the actual payload length
		frame = append(frame, byte(payloadLength))
	} else if payloadLength < 65536 { // the following 16 bits is the payload length
		frame = append(frame, byte(126))
		frame = frame[:4]
		binary.BigEndian.PutUint16(frame[2:4], uint16(payloadLength))
	} else { // the following 64 bits is the payload length
		frame = append(frame, byte(127))
		frame = frame[:10]
		binary.BigEndian.PutUint64(frame[2:10], uint64(payloadLength))
	}

	// TODO: wonder if we should add data to frame (requires a copy), or just call
	// c.underlying.Write twice. the former is probably faster however

	// header size
	hsize := len(frame)
	// expand the slice length to its needed size (header size + payloadLength)
	frame = frame[:hsize+payloadLength]
	copy(frame[hsize:], data) // top data off after the header

	_, err := c.underlying.Write(frame)
	if err != nil {
		return ErrConnectionWrite
	}
	return nil
}

// Ping writes a ping frame to the connection. If a nil context is specified,
// it will default to five seconds. If no response is reached within
// the duration, it will return false. It may return an error if
// there is an issue writing to the connection.
//
// If there is a ping that has not recieved a pong yet, calling this
// function will NOT write another ping frame, but will block until it recieves
// a pong.
func (c *Conn) Ping(ctx context.Context) (bool, error) {

	c.pingMx.Lock()
	if c.pingCtx == nil {
		var cancel context.CancelFunc
		if ctx == nil {
			ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
		} else {
			ctx, cancel = context.WithCancel(ctx)
		}
		c.pingCtx = ctx
		c.pingCancel = cancel
		err := c.Write(&Message{
			Type: MessagePing,
			Data: []byte{},
		})
		if err != nil {
			return false, err
		}
	}
	c.pingMx.Unlock()

	<-c.pingCtx.Done()

	switch ctx.Err() {
	case context.Canceled:
		return true, nil
	case context.DeadlineExceeded: // a pong was not recieved in a timely manner
		return false, nil
	}
	return false, nil
}
