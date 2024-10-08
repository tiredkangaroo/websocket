package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"strings"
)

// AcceptHTTP handles a WebSocket HTTP request from the client. It may return
// an error if the HTTP request is not a WebSocket connection, the WebSocket
// version is not supported, the Sec-WebSocket-Key is not provided, or hijacking
// the underlying connection fails.
func AcceptHTTP(w http.ResponseWriter, r *http.Request) (*Conn, WebsocketError) {
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
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, errorf(KEY_NOT_PROVIDED)
	}

	// https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API/Writing_WebSocket_servers#server_handshake_response
	concatenatedKey := strings.TrimSpace(key) + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	hashedCKey := sha1.Sum([]byte(concatenatedKey))
	acceptKey := base64.StdEncoding.EncodeToString(hashedCKey[:])

	w.Header().Set("Upgrade", "websocket")
	w.Header().Set("Connection", "Upgrade")
	w.Header().Set("Sec-WebSocket-Accept", acceptKey)
	w.WriteHeader(101)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errorf(HTTP_HIJACKING_FAILED)
	}

	conn, _, err := hijacker.Hijack()
	if err != nil {
		return nil, errorf(HTTP_HIJACKING_FAILED)
	}

	return &Conn{underlying: conn}, nil
}
