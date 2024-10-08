package extended

import "websocket"

// OnMessage takes in a WebSocket connection and a function with parameter
// of a WebSocket Message and calls the function on message. It will stop calling
// the function once the connection has been closed.
func OnMessage(conn *websocket.Conn, f func(*websocket.Message)) {
	go func() {
		for {
			msg, err := conn.Read()
			if err != nil {
				if err.Kind() == websocket.CONNECTION_CLOSED {
					return
				}
			}
			f(msg)
		}
	}()
}
