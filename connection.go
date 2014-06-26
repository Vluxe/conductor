// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Some code currently borrowed from example chat program of
// https://github.com/gorilla/websocket

package conductor

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Unless the plan changes, we are going to be using this for client and peer connections
func serveWs(w http.ResponseWriter, r *http.Request) {
	log.Println("serving ws")
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}
	log.Println("connection upgraded!")
}
