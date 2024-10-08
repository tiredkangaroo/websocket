package main

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"websocket"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.AcceptHTTP(w, r)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		for {
			msg, err := conn.Read()
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			fmt.Printf("-> %s\n", msg)

			fullmsg := &websocket.Message{
				Type: websocket.MessageText,
				Data: []byte(hex.EncodeToString(msg.Data)),
			}
			err = conn.Write(fullmsg)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			fmt.Printf("<- %s\n", fullmsg)

			fmt.Printf("pinging server... ")
			alive, err := conn.Ping(nil)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			fmt.Printf("alive: %t\n", alive)

		}
	})

	http.ListenAndServe(":8000", nil)
}
