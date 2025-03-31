package websocket_test

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tiredkangaroo/websocket"
)

type MockResponseWriterNoHijack struct {
	httptest.ResponseRecorder
}

type MockResponseWriterHijack struct {
	httptest.ResponseRecorder
}

var hijack_mock_conn = new(MockNetConn)
var hijack_bufioreadwriter = bufio.NewReadWriter(bufio.NewReader(&hijack_mock_conn.buf), bufio.NewWriter(&hijack_mock_conn.buf))
var hijack_net_conn = net.Conn(hijack_mock_conn)

func (m *MockResponseWriterHijack) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return hijack_net_conn, hijack_bufioreadwriter, nil
}

// TestAcceptHTTPSuccess checks if a valid WebSocket request is correctly accepted.
func TestAcceptHTTPSuccess(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==") // base64-encoded test key

	rec := new(MockResponseWriterHijack)

	conn, err := websocket.AcceptHTTP(rec, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if conn == nil {
		t.Fatal("expected a valid WebSocket connection, got nil")
	}
	if rec.Result().StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("expected status %d, got %d", http.StatusSwitchingProtocols, rec.Result().StatusCode)
	}
	if rec.Result().Header.Get("Sec-WebSocket-Accept") != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		fmt.Println(rec.Result().Header.Get("Sec-WebSocket-Accept"))
		t.Errorf("bad accept key")
	}
}

// TestAcceptHTTPNotWebSocket checks if a non-WebSocket request is rejected.
func TestAcceptHTTPNotWebSocket(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost/ws", nil)
	req.Header.Set("Upgrade", "not-websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	rec := new(MockResponseWriterHijack)

	conn, err := websocket.AcceptHTTP(rec, req)
	if err == nil || conn != nil {
		t.Fatal("expected error for non-WebSocket request, got none")
	}
}

// TestAcceptHTTPVersionNotSupported checks if an unsupported WebSocket version is rejected.
func TestAcceptHTTPVersionNotSupported(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "14") // unsupported version
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	rec := new(MockResponseWriterHijack)

	conn, err := websocket.AcceptHTTP(rec, req)
	if err == nil || conn != nil {
		t.Fatal("expected error for unsupported WebSocket version, got none")
	}
}

// TestAcceptHTTPKeyNotProvided checks if a request without a WebSocket key is rejected.
func TestAcceptHTTPKeyNotProvided(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	// Missing Sec-WebSocket-Key header

	rec := new(MockResponseWriterHijack)

	conn, err := websocket.AcceptHTTP(rec, req)
	if err == nil || conn != nil {
		t.Fatal("expected error for missing WebSocket key, got none")
	}
}

// TestAcceptHTTPHijackingFailed checks if an error is returned when hijacking fails.
func TestAcceptHTTPHijackingFailed(t *testing.T) {
	// Use a ResponseWriter that doesn't support hijacking
	req, _ := http.NewRequest("GET", "http://localhost/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	rec := new(MockResponseWriterNoHijack) // cannot hijack this conn

	conn, err := websocket.AcceptHTTP(rec, req)
	if err == nil || conn != nil {
		t.Fatal("expected error due to hijacking failure, got none")
	}
}
