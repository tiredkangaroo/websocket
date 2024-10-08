package websocket

import (
	"context"
	"encoding/binary"
	"log/slog"
	"net"
	"time"
)

// Conn represents a WebSocket connection.
type Conn struct {
	underlying net.Conn
	closed     bool
}

// Close marks the connection as closed and closes the underlying
// connection. It may return an error if there is an issue closing
// the underlying connection.
func (c *Conn) Close() error {
	c.closed = true
	return c.underlying.Close()
}

// Read reads a WebSocket frame from the underlying connection. If there
// is an issue reading the frame or the frame is malformed, it may return
// an error.
func (c Conn) Read() (*Message, WebsocketError) {
	message := new(Message)

	header := make([]byte, 2) // includes fin, rsv1, rsv2, rsv3, and opcode

	n, err := c.underlying.Read(header)
	if err != nil {
		return nil, manageReadError(err)
	}
	if n != 2 {
		return nil, errorf(MALFORMED_FRAME, "read 0 bytes, expected 1")
	}

	fin := (header[0] & 0x80) != 0

	// FIXME: fragmented frames are not supported
	if !fin { // if frame is fragmented (0 means fragmented, 1 means final)
		return nil, errorf(MALFORMED_FRAME, "fragmented frames are not supported")
	}

	rsv1 := (header[0] & 0x40) != 0
	rsv2 := (header[0] & 0x20) != 0
	rsv3 := (header[0] & 0x10) != 0

	if (rsv1 || rsv2 || rsv3) == true { // for extensions
		return nil, errorf(MALFORMED_FRAME, "rsv1, rsv2, and/or rsv3 are specified")
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
		message.Type = MessagePong
	default:
		return nil, errorf(MALFORMED_FRAME, "unknown opcode")
	}

	// payload length
	payloadLength := int(header[1] & 0x7F) // extenstion data + application data in bytes
	switch payloadLength {
	case 126: // the following 16 bits (or 2 bytes) is the uint payload length
		extendedPayloadLen := make([]byte, 2)
		_, err = c.underlying.Read(extendedPayloadLen)
		if err != nil {
			return nil, manageReadError(err)
		}
		payloadLength = int(binary.BigEndian.Uint16(extendedPayloadLen))
	case 127: // the following 64 bits (or 8 bytes) is the uint payload length
		extendedPayloadLen := make([]byte, 8)
		_, err = c.underlying.Read(extendedPayloadLen)
		if err != nil {
			return nil, manageReadError(err)
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
			return nil, manageReadError(err)
		}
	}

	// the actual payload
	payload := make([]byte, payloadLength)
	_, err = c.underlying.Read(payload)
	if err != nil {
		return nil, manageReadError(err)
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
func (c Conn) Write(message *Message) WebsocketError {
	messageType := message.Type
	data := message.Data

	frame := []byte{}
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
		extendedPayloadLength := make([]byte, 2)
		binary.BigEndian.PutUint16(extendedPayloadLength, uint16(payloadLength))
		frame = append(frame, extendedPayloadLength...)
	} else { // the following 64 bits is the payload length
		frame = append(frame, byte(127))
		extendedPayloadLength := make([]byte, 8)
		binary.BigEndian.PutUint64(extendedPayloadLength, uint64(payloadLength))
		frame = append(frame, extendedPayloadLength...)
	}

	frame = append(frame, data...)

	_, err := c.underlying.Write(frame)

	return manageWriteError(err)
}

// Ping writes a ping frame to the connection. If a nil context is specified,
// it will default to five seconds. If no response is reached within
// the duration, it will return false. It may return an error if
// there is an issue writing to the connection.
func (c Conn) Ping(ctx context.Context) (bool, WebsocketError) {
	err := c.Write(&Message{
		Type: MessagePing,
		Data: []byte{},
	})
	if err != nil {
		return false, err
	}

	var cancel context.CancelFunc
	if ctx == nil {
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}

	go func() {
		for {
			msg, err := c.Read()
			if err != nil {
				break
			}
			if msg.Type == MessagePong {
				cancel()
				break
			}
		}
	}()

	<-ctx.Done()
	switch ctx.Err() {
	case context.Canceled:
		return true, nil
	case context.DeadlineExceeded:
		return false, nil
	}
	return false, nil
}
