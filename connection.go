// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// some code borrowed from example chat program of
// https://github.com/gorilla/websocket
package conductor

import (
	"encoding/json"
	"fmt"
	"github.com/acmacalister/skittles"
	"github.com/gorilla/websocket"
	"log"
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

// connection is an middleman between the websocket connection and the hub.
type connection struct {
	ws       *websocket.Conn
	send     chan Message
	channels []string
	peer     bool
	name     string
	token    string
}

// broadcasting message
type broadcastWriter struct {
	conn    *connection
	message *Message
	peer    bool
}

// readPump pumps messages from the websocket connection to the hub.
func (c *connection) readPump(server *Server) {
	defer c.closeConnection(server)

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

		message.Name = c.name
		if message.OpCode == PeerBindOpCode && c.peer {
			server.connectToPeer(message.Body)
		} else if message.OpCode == ServerOpCode {
			if server.ServerQuery != nil {
				c.send <- server.ServerQuery.QueryHandler(message, c.token)
			}
		} else if message.OpCode == InviteOpCode {
			if c.canWrite(&message, server) {
				server.hub.invite <- broadcastWriter{conn: c, message: &message, peer: false}
			}
		} else {
			if message.OpCode == BindOpCode {
				c.bind(&message, server)
			} else if message.OpCode == UnBindOpCode {
				c.unbind(&message, server)
			}
			if server.Notification != nil && message.OpCode == WriteOpCode {
				server.Notification.PersistentHandler(message, c.token)
			}
			if c.canWrite(&message, server) {
				server.hub.broadcast <- broadcastWriter{conn: c, message: &message, peer: false}
			}
		}
	}
}

// bind to a channel.
func (c *connection) bind(message *Message, server *Server) {
	authStatus := true
	if !c.peer && server.Auth != nil {
		authStatus = server.Auth.ChannelAuthHandler(*message, c.token)
	}
	if authStatus {
		addChannel := true
		for _, channel := range c.channels {
			if channel == message.ChannelName {
				addChannel = false
				break
			}
		}
		if addChannel {
			server.hub.bind <- broadcastWriter{conn: c, message: message, peer: false}
			c.channels = append(c.channels, message.ChannelName)
		}
	} else {
		log.Println(skittles.BoldRed(fmt.Sprintf("%s: was unable to connect to the channel", c.name)))
		c.closeConnection(server)
	}
}

//unbind from a channel
func (c *connection) unbind(message *Message, server *Server) {
	authStatus := true
	if !c.peer && server.Auth != nil {
		authStatus = server.Auth.ChannelAuthHandler(*message, c.token)
	}

	if authStatus {
		for i, channel := range c.channels {
			if channel == message.ChannelName {
				c.channels = append(c.channels[:i], c.channels[i+1:]...)
				break
			}
		}
		server.hub.unbind <- broadcastWriter{conn: c, message: message, peer: false}
	} else {
		log.Println(skittles.BoldRed(fmt.Sprintf("%s: was unable to connect to the channel", c.name)))
		c.closeConnection(server)
	}
}

//check and make sure we can write this
func (c *connection) canWrite(message *Message, server *Server) bool {
	if c.peer {
		return true
	}
	authStatus := true
	if !c.peer && server.Auth != nil {
		authStatus = server.Auth.MessageAuthHandler(*message, c.token)
	}
	//check if this a channel the client is bound to
	if authStatus {
		authStatus = false
		for _, channel := range c.channels {
			if channel == message.ChannelName {
				return true
			}
		}
	}
	return authStatus
}

//closes the connection
func (c *connection) closeConnection(server *Server) {
	server.hub.unregister <- c
	c.ws.Close()
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
				authStatus = server.Auth.MessageAuthHandler(message, c.token)
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
