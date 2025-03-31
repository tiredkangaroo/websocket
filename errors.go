package websocket

import (
	"errors"
)

var (
	// ErrRequestNotWebsocket indicates that the HTTP request provided does not specify
	// instructions for a WebSocket upgrade.
	ErrRequestNotWebsocket = errors.New("the request does not specify a websocket upgrade")
	// ErrVersionNotSupported indicates that the version provided in the request is not
	// supported. Currently supported versions: 13 (as per the RFC).
	ErrVersionNotSupported = errors.New("the request specifies an unsupported version")
	// ErrKeyNotProvided indicates that there is no Sec-WebSocket-Key passed in by the
	// client.
	ErrKeyNotProvided = errors.New("the request does not specify a Sec-WebSocket-Key")
	// ErrHijackFailed indicates an error with hijacking the underlying connection from
	// http.ResponseWriter.
	ErrHijackFailed = errors.New("unable to hijack the http connection")
	// ErrConnectionRead indicates an error reading from the underlying connection.
	ErrConnectionRead = errors.New("reading from the underlying connection failed")
	// ErrConnectionWrite indicates an error writing to the underlying connection.
	ErrConnectionWrite = errors.New("writing to the underlying connection failed")
	// ErrConnectionClosed indicates that the underlying connection is closed. This connection
	// cannot be read from or written to.
	ErrConnectionClosed = errors.New("connection is closed")
	// ErrMalformedFrame indicates that the server recieved an unexpectedly formed frame.
	ErrMalformedFrame = errors.New("websocket frame is malformed")
)
