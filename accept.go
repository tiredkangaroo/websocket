package websocket

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
)

// AcceptHTTP handles a WebSocket HTTP request from the net/http client. It may return
// an error if the HTTP request is not a WebSocket connection, the WebSocket
// version is not supported, the Sec-WebSocket-Key is not provided, or hijacking
// the underlying connection fails.
func AcceptHTTP(w http.ResponseWriter, r *http.Request) (*Conn, Error) {
	// verify request is for a WebSocket connection and get the Sec-Websocket-Key
	// https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API/Writing_WebSocket_servers#client_handshake_request
	upgrade := r.Header.Get("Upgrade")
	if upgrade != "websocket" {
		return nil, errorf(REQUEST_NOT_WEBSOCKET)
	}
	connection := r.Header.Get("Connection")
	if connection != "Upgrade" {
		return nil, errorf(REQUEST_NOT_WEBSOCKET)
	}
	version := r.Header.Get("Sec-WebSocket-Version")
	if version != "13" {
		return nil, errorf(VERSION_NOT_SUPPORTED)
	}
	key := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		return nil, errorf(KEY_NOT_PROVIDED)
	}

	// developing the Sec-WebSocket-Accept key
	// https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API/Writing_WebSocket_servers#server_handshake_response
	concatenatedKey := strings.TrimSpace(key) + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	hashedCKey := sha1.Sum([]byte(concatenatedKey))
	acceptKey := base64.StdEncoding.EncodeToString(hashedCKey[:])

	// set the server WebSocket Handshake Response headers
	w.Header().Set("Upgrade", "websocket")
	w.Header().Set("Connection", "Upgrade")
	w.Header().Set("Sec-WebSocket-Accept", acceptKey)
	w.WriteHeader(101)

	// now that the handshake is done, we now have a WebSocket connection expected
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errorf(HTTP_HIJACKING_FAILED)
	}

	conn, _, err := hijacker.Hijack()
	if err != nil {
		return nil, errorf(HTTP_HIJACKING_FAILED)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Conn{underlying: conn, rmx: sync.Mutex{}, wmx: sync.Mutex{}, ctx: ctx, cancel: cancel, closed: false}, nil
}
