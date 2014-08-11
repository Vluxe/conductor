// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// some code borrowed from example chat program of
// https://github.com/gorilla/websocket
package conductor

import (
	"encoding/json"
	"log"
	"time"

	"github.com/acmacalister/skittles"
	"github.com/gorilla/websocket"
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

// connection is an middleman between the websocket connection and the hub.
type connection struct {
	ws       *websocket.Conn
	send     chan Message
	channels []string
	peer     bool
	peerName string
}

// broadcasting message
type broadcastWriter struct {
	conn    *connection
	message Message
	peer    bool
}

// readPump pumps messages from the websocket connection to the hub.
func (c *connection) readPump(server *Server) {
	defer func() {
		server.hub.unregister <- c
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
		// Peer message wanting to connect to a peer
		if message.OpCode == Peer {
			log.Println("peer url:", message.Body)
			//c.bind(message, server)
			server.connectToPeer(message.Body)
		} else {

			if message.OpCode == Bind {
				c.bind(message, server)
			}
			server.Notification.PersistentHandler()
			server.hub.broadcast <- broadcastWriter{conn: c, message: message, peer: false}
		}
	}
}

// bind to a channel.
func (c *connection) bind(message Message, server *Server) {
	authStatus := true
	if server.Auth != nil {
		authStatus = server.Auth.ChannelAuthHandler()
	}

	if authStatus {
		addChannel := true
		for _, channel := range c.channels {
			if channel == message.ChannelName {
				addChannel = false
			}
		}
		if addChannel {
			server.hub.bind <- broadcastWriter{conn: c, message: message, peer: false}
			c.channels = append(c.channels, message.ChannelName)
		}
	} else {
		log.Println("what do we do now?")
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *connection) writePump(server *Server) {
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
			if server.Auth != nil {
				authStatus = server.Auth.MessageAuthHandler()
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

// write writes a message with the given message type and payload.
func (c *connection) write(mt int, payload Message) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	buf, _ := json.Marshal(payload)
	return c.ws.WriteMessage(mt, buf)
}
