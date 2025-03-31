package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"unsafe"
)

// sec_websocket_key_noncesize is the size of Sec-WebSocket-Key.
// From https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Sec-WebSocket-Key:
// The key for this request to upgrade. This is a randomly selected 16-byte nonce that has been
// base64-encoded and isomorphic encoded.  The user agent adds this when initiating the WebSocket connection.
//
// A 16-byte nonce base-64 encoded is 24 characters long.
const sec_websocket_key_noncesize = 24
const websocket_uuid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// AcceptHTTP handles a WebSocket HTTP request from the net/http client. It may return
// an error if the HTTP request is not a WebSocket connection, the WebSocket
// version is not supported, the Sec-WebSocket-Key is not provided, or hijacking
// the underlying connection fails.
func AcceptHTTP(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	// verify request is for a WebSocket connection and get the Sec-Websocket-Key
	// https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API/Writing_WebSocket_servers#client_handshake_request
	upgrade := r.Header.Get("Upgrade")
	if upgrade != "websocket" {
		return nil, ErrRequestNotWebsocket
	}
	version := r.Header.Get("Sec-WebSocket-Version")
	if version != "13" {
		return nil, ErrVersionNotSupported
	}
	key := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		return nil, ErrKeyNotProvided
	}

	// developing the Sec-WebSocket-Accept key
	// https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API/Writing_WebSocket_servers#server_handshake_response
	var keyConcat []byte
	var acceptKey [28]byte

	if len(key) == sec_websocket_key_noncesize { // fast path
		keyConcatArray := [sec_websocket_key_noncesize + len(websocket_uuid)]byte{}
		copy(keyConcatArray[:], key+websocket_uuid)
		keyConcat = keyConcatArray[:]
	} else { // slow path
		keyConcat = []byte(key + websocket_uuid)
	}

	hashedCKey := sha1.Sum(keyConcat[:])
	base64.StdEncoding.Encode(acceptKey[:], hashedCKey[:])

	// set the server WebSocket Handshake Response headers
	w.Header().Set("Upgrade", "websocket")
	w.Header().Set("Connection", "Upgrade")
	w.Header().Set("Sec-WebSocket-Accept", unsafe.String(&acceptKey[0], len(acceptKey)))
	w.WriteHeader(101)

	// now that the handshake is done, we now have a WebSocket connection expected
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, ErrHijackFailed
	}

	conn, _, err := hijacker.Hijack()
	if err != nil {
		return nil, ErrHijackFailed
	}

	return &Conn{underlying: conn, rmx: sync.Mutex{}, wmx: sync.Mutex{}, closed: false}, nil
}
