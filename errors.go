package websocket

import (
	"fmt"
	"net"
)

const (
	// REQUEST_NOT_WEBSOCKET indicates that the HTTP request provided does not specify
	// instructions for a WebSocket upgrade.
	REQUEST_NOT_WEBSOCKET = "the request does not specify a websocket upgrade"
	// VERSION_NOT_SUPPORTED indicates that the version provided in the request is not
	// supported. Currently supported versions: 13.
	VERSION_NOT_SUPPORTED = "the request specifies an unsupported version"
	// KEY_NOT_PROVIDED indicates that there is no Sec-WebSocket-Key passed in by the
	// client.
	KEY_NOT_PROVIDED = "the request does not specify a Sec-WebSocket-Key"
	// HTTP_HIJACKING_FAILED indicates an error with hijacking the underlying connection
	// from the hijacker.
	HTTP_HIJACKING_FAILED = "unable to hijack the http connection"
	// CONNECTION_READ_ERROR indicates an error reading the underlying connection. This
	// differs from the CONNECTION_CLOSED error.
	CONNECTION_READ_ERROR = "reading from the connection failed: %s"
	// CONNECTION_WRITE_ERROR indicates an error reading the underlying connection. This
	// differs from the CONNECTION_CLOSED error.
	CONNECTION_WRITE_ERROR = "writing to the connection failed: %s"
	// CONNECTION_CLOSED indicates that the underlying connection is closed. This connection
	// cannot be read from or written to.
	CONNECTION_CLOSED = "connection is closed"
	// MALFORMED_FRAME indicates that the server recieved an unexpectedly formed frame.
	MALFORMED_FRAME = "websocket frame is malformed: %s"
)

// WebsocketError implements the error interface and provides
// the Kind of the error.
type WebsocketError interface {
	Kind() string
	Error() string
}

type err struct {
	kind string
	err  string
}

func (e err) Kind() string {
	return e.Kind()
}

func (e err) Error() string {
	return e.err
}

func errorf(kind string, a ...any) WebsocketError {
	return err{
		kind: kind,
		err:  fmt.Sprintf(kind, a...),
	}
}

// isUseOfClosedNetworkConnectionError determines whether or not the error
// passed in is a use of closed network connection error.
func isUseOfClosedNetworkConnectionError(err error) bool {
	return err == net.ErrClosed
	// return strings.Contains(err.Error(), "use of closed network connection")
}

// manageReadError look at the error after reading and either returns a
// CONNECTION_CLOSED or a CONNECTION_READ_ERROR. If the error passed in is nil,
// the function will return nil as to avoid a nil pointer dereference.
func manageReadError(err error) WebsocketError {
	if err == nil {
		return nil
	}
	if isUseOfClosedNetworkConnectionError(err) {
		return errorf(CONNECTION_CLOSED)
	} else {
		return errorf(CONNECTION_READ_ERROR, err.Error())
	}
}

// manageWriteError look at the error after reading and either returns a
// CONNECTION_CLOSED or a CONNECTION_WRITE_ERROR. If the error passed in is nil,
// the function will return nil as to avoid a nil pointer dereference.
func manageWriteError(err error) WebsocketError {
	if err == nil {
		return nil
	}
	if isUseOfClosedNetworkConnectionError(err) {
		return errorf(CONNECTION_CLOSED)
	} else {
		return errorf(CONNECTION_WRITE_ERROR, err.Error())
	}
}
