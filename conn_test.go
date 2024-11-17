package websocket_test

import (
	"bytes"
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"
	"websocket"
)

// MockNetConn implements net.Conn for testing.
type MockNetConn struct {
	buf    bytes.Buffer
	closed bool
	mutex  sync.Mutex
}

func (m *MockNetConn) Read(p []byte) (int, error) {
	if m.closed {
		return 0, io.EOF
	}
	return m.buf.Read(p)
}

func (m *MockNetConn) Write(p []byte) (int, error) {
	if m.closed {
		return 0, io.EOF
	}
	return m.buf.Write(p)
}

func (m *MockNetConn) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.closed = true
	return nil
}

func (m *MockNetConn) LocalAddr() net.Addr {
	return nil
}

func (m *MockNetConn) RemoteAddr() net.Addr {
	return nil
}

func (m *MockNetConn) SetDeadline(_ time.Time) error {
	return nil
}

func (m *MockNetConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (m *MockNetConn) SetWriteDeadline(_ time.Time) error {
	return nil
}

func TestContext(t *testing.T) {
	mockConn := &MockNetConn{}
	conn := websocket.From(mockConn)
	if conn.Context() == nil {
		t.Fatalf("expected Context() not be nil")
	}
}

func TestClose(t *testing.T) {
	mockConn := &MockNetConn{}
	conn := websocket.From(mockConn)

	err := conn.Close()
	if err != nil {
		t.Fatalf("Expected no error from Close(), got %v", err)
	}
	if !mockConn.closed {
		t.Fatalf("Expected underlying connection to be closed")
	}
	if conn.Context().Err() != context.Canceled {
		t.Fatalf("Expected context to be canceled")
	}
}

func TestRead_ClosedConnection(t *testing.T) {
	mockConn := &MockNetConn{}
	conn := websocket.From(mockConn)
	conn.Close()

	_, err := conn.Read()
	if err == nil || err.Kind() != websocket.CONNECTION_CLOSED {
		t.Fatalf("Expected CONNECTION_CLOSED error, got %v", err)
	}
}

func TestRead_MalformedFrame(t *testing.T) {
	mockConn := &MockNetConn{}
	conn := websocket.From(mockConn)

	// unsupported opcode (0x3):
	mockConn.buf.Write([]byte{0x83, 0x00})
	_, err := conn.Read()
	if err == nil || err.Kind() != websocket.MALFORMED_FRAME {
		t.Fatalf("Expected MALFORMED_FRAME error, got %v", err)
	}
}

func TestWrite_MessageText(t *testing.T) {
	mockConn := &MockNetConn{}
	conn := websocket.From(mockConn)

	message := &websocket.Message{Type: websocket.MessageText, Data: []byte("Hello")}
	err := conn.Write(message)
	if err != nil {
		t.Fatalf("Expected no error from Write(), got %v", err)
	}

	expected := []byte{0x81, 0x05, 'H', 'e', 'l', 'l', 'o'}
	if !bytes.Equal(mockConn.buf.Bytes(), expected) {
		t.Fatalf("Expected %v, got %v", expected, mockConn.buf.Bytes())
	}
}

func TestRead_MessageText(t *testing.T) {
	mockConn := &MockNetConn{}
	conn := websocket.From(mockConn)
	mockConn.buf.Write([]byte("this is a hello world!"))

	expected := []byte{0x81, 0x16}
	expected = append(expected, []byte("this is a hello world!")...)

	msg, err := conn.Read()
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(msg.Data, expected) {
		t.Fatalf("Expected %v, got %v", expected, mockConn.buf.Bytes())
	}
}

func TestPing_PongReceived(t *testing.T) {
	mockConn := &MockNetConn{}
	conn := websocket.From(mockConn)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		pongReceived, err := conn.Ping(ctx)
		if err != nil {
			t.Errorf("Unexpected error from Ping: %v", err)
		}
		if !pongReceived {
			t.Errorf("Expected pong response")
		}
	}()
	// Simulate a pong response
	time.Sleep(100 * time.Millisecond)
	cancel()

	<-done
}

func TestPing_Timeout(t *testing.T) {
	mockConn := &MockNetConn{}
	conn := websocket.From(mockConn)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	pongReceived, err := conn.Ping(ctx)
	if err != nil {
		t.Fatalf("Expected no error from Ping, got %v", err)
	}
	if pongReceived {
		t.Fatalf("Expected Ping to timeout, but got pong response")
	}
}

func TestRead_UnmaskPayload(t *testing.T) {
	mockConn := &MockNetConn{}
	conn := websocket.From(mockConn)

	// fin: 1, opcode: test, masking key: present, payload size: 4, masking key: 1, 2, 3, 4, payload(masked("test"))
	mockConn.buf.Write([]byte{0x81, 0x84, 0x01, 0x02, 0x03, 0x04, 0x75, 0x67, 0x70, 0x70})

	message, err := conn.Read()
	if err != nil {
		t.Fatalf("Expected no error from Read, got %v", err)
	}

	expectedData := []byte("test")
	if !bytes.Equal(message.Data, expectedData) {
		t.Fatalf("Expected data %v, got %v", expectedData, message.Data)
	}
}
