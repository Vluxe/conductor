// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// code borrowed from example chat program of
// https://github.com/gorilla/websocket

package conductor

import (
	"github.com/acmacalister/skittles"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 500 // Don't leave me like this!!
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var author ConductorAuth

// We need some channels up in here.
// type channel struct {
// 	UUID string
// 	Name string
// }

// connection is an middleman between the websocket connection and the hub.
type connection struct {
	ws       *websocket.Conn
	send     chan []byte
	channels []string
}

// broadcasting message
type broadcastWriter struct {
	conn        *connection
	message     []byte
	channelUUID string
}

// interface for auth methods
type ConductorAuth interface {
	InitalAuthHander(r *http.Request) bool
	ChannelAuthHander() bool
	MessageAuthHander() bool
}

// readPump pumps messages from the websocket connection to the hub.
func (c *connection) readPump() {
	defer func() {
		h.unregister <- c
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		var message Message
		err := c.ws.ReadJSON(&message)
		if err != nil {
			log.Println(skittles.BoldRed(err))
			break
		}
		h.broadcast <- broadcastWriter{conn: c, message: []byte(message.Body), channelUUID: message.ChannelName}
	}
}

// write writes a message with the given message type and payload.
func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket connection.
func (c *connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	log.Println("writing message")
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// serverWs handles webocket requests from the peer.
func serveWs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	authStatus := true
	if author != nil {
		authStatus = author.InitalAuthHander(r)
	}

	if authStatus {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				log.Println(err)
			}
			return
		}
		var channels []string
		channels = append(channels, "hello")
		c := &connection{send: make(chan []byte, 256), ws: ws, channels: channels}
		h.register <- c
		go c.writePump()
		c.readPump()
	} else {
		http.Error(w, "Failed ", 401)
	}
}

// Pass in your implemented type of Auth interface.
func ServeAuth(a ConductorAuth) {
	author = a
}
