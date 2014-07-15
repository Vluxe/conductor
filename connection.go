// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// code borrowed from example chat program of
// https://github.com/gorilla/websocket
package conductor

import (
	"encoding/json"
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

var (
	author     ConductorAuth
	persistent ConductorPersistent
)

// We need some channels up in here.
// type channel struct {
// 	UUID string
// 	Name string
// }

// connection is an middleman between the websocket connection and the hub.
type connection struct {
	ws       *websocket.Conn
	send     chan Message
	channels []string
}

// broadcasting message
type broadcastWriter struct {
	conn    *connection
	message Message
}

// interface for auth methods
type ConductorAuth interface {
	InitalAuthHander(r *http.Request) bool
	ChannelAuthHander() bool
	MessageAuthHander() bool
}

// pass in your implemented type of ConductorAuth interface.
func ServeAuth(a ConductorAuth) {
	author = a
}

type ConductorPersistent interface {
	PersistentHandler()
}

// pass in your implemented type of ConductorPersistent interface.
func ServePersistent(p ConductorPersistent) {
	persistent = p
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
		if message.OpCode == Bind {
			c.bind(message)
		}

		persistent.PersistentHandler()
		h.broadcast <- broadcastWriter{conn: c, message: message}
	}
}

// bind to a channel.
func (c *connection) bind(message Message) {
	authStatus := true
	if author != nil {
		authStatus = author.ChannelAuthHander()
	}

	if authStatus {
		addChannel := true
		for _, channel := range c.channels {
			if channel == message.ChannelName {
				addChannel = false
			}
		}
		if addChannel {
			h.register <- c
			c.channels = append(c.channels, message.ChannelName)
		}
	} else {
		log.Println("what do we do now?")
	}
}

// write writes a message with the given message type and payload.
func (c *connection) write(mt int, payload Message) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	buf, _ := json.Marshal(payload)
	return c.ws.WriteMessage(mt, buf)
}

// writePump pumps messages from the hub to the websocket connection.
func (c *connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, Message{})
				return
			}
			authStatus := true
			if author != nil {
				authStatus = author.MessageAuthHander()
			}
			err := c.write(websocket.TextMessage, message)
			if err != nil && authStatus {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, Message{}); err != nil {
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
		c := &connection{send: make(chan Message, 256), ws: ws, channels: channels}
		h.register <- c
		go c.writePump()
		c.readPump()
	} else {
		http.Error(w, "Failed ", 401)
	}
}
