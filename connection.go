// Copyright 2013/2014 The Gorilla WebSocket & Conductor Authors. All rights reserved.
// some code borrowed from example chat program of
// https://github.com/gorilla/websocket
// Gorilla under BSD-style. See it's LICENSE file for more info.
// Conductor under Apache v2. License can be found in the LICENSE file.

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

// broadcastWriter is type for wrapping a connection,
// message and peer setting into something concrete to
// send to the hub.
type broadcastWriter struct {
	conn    *connection
	message *Message
	peer    bool
}

// readPump sends messages from the connection to the hub.
// param: server - Takes a server struct to run it's callbacks.
func (c *connection) readPump(server *Server) {
	defer c.closeConnection(server)

	// Setup our connection's websocket ping/pong handlers from our const values.
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	// This is our blocking loop. Hence us starting it in it's own goroutine.
	// It processes all the read messages for this connection.
	for {
		var message Message
		err := c.ws.ReadJSON(&message)
		if err != nil {
			log.Println(skittles.BoldRed(err))
			break
		}

		message.Name = c.name
		switch message.OpCode {
		case PeerBindOpCode: // message for connecting peers.
			c.peerBindOp(server, &message)
		case ServerOpCode: // message from a "client" to run our serverQuery callback.
			c.serverOp(server, &message)
		case InviteOpCode: // message when an invitation to join a channel is sent.
			c.inviteOp(server, &message)
		case BindOpCode: // message to bind to a channel.
			c.bindOp(server, &message)
		case UnBindOpCode: // message to unbind to channel.
			c.unbindOp(server, &message)
		case WriteOpCode: // message to writes to channel.
			c.writeOp(server, &message)
		default: // broadcast message.
			c.broadcastMessage(server, &message)
		}
	}
}

// peerBindOp when a connection needs to connect to a peer.
func (c *connection) peerBindOp(server *Server, message *Message) {
	if c.peer {
		server.connectToPeer(message.Body)
	}
}

// serverOp when a connection needs to execute query handler callback.
func (c *connection) serverOp(server *Server, message *Message) {
	if server.ServerQuery != nil {
		c.send <- server.ServerQuery.QueryHandler(*message, c.token)
	}
}

// inviteOp when a connection needs to send an invitation out.
func (c *connection) inviteOp(server *Server, message *Message) {
	if server.Notification != nil {
		server.Notification.InviteHandler(*message, c.token)
	}
	server.hub.invite <- broadcastWriter{conn: c, message: message, peer: false}
}

// bindOp when a connection needs to bind to a channel.
func (c *connection) bindOp(server *Server, message *Message) {
	c.bind(message, server)
	if server.Notification != nil {
		server.Notification.BindHandler(*message, c.token)
	}
	c.broadcastMessage(server, message)
}

// unbindOp when a connection needs to unbind from a channel.
func (c *connection) unbindOp(server *Server, message *Message) {
	c.unbind(message, server)
	if server.Notification != nil {
		server.Notification.UnBindHandler(*message, c.token)
	}
	c.broadcastMessage(server, message)
}

// writeOp when a connection needs to write to a channel.
func (c *connection) writeOp(server *Server, message *Message) {
	if server.Notification != nil {
		server.Notification.PersistentHandler(*message, c.token)
	}
	c.broadcastMessage(server, message)
}

// broadcastMessage when a connection needs to send a message out to the hub.
func (c *connection) broadcastMessage(server *Server, message *Message) {
	if c.canWrite(message, server) {
		server.hub.broadcast <- broadcastWriter{conn: c, message: message, peer: false}
	}
}

// checkAuthStatus to see if the connection is allowed to send to channel/message.
func (c *connection) checkAuthStatus(server *Server, message *Message, channel bool) bool {
	if !c.peer && server.Auth != nil {
		if channel {
			return server.Auth.ChannelAuthHandler(*message, c.token)
		} else {
			return server.Auth.MessageAuthHandler(*message, c.token)
		}
	}
	return true
}

// bind to a channel.
func (c *connection) bind(message *Message, server *Server) {
	authStatus := c.checkAuthStatus(server, message, true)
	if authStatus {
		addChannel := true
		for _, channel := range c.channels { // check to see if we are already bound to channel.
			if channel == message.ChannelName {
				addChannel = false
				break
			}
		}

		if addChannel { // if we aren't add the channel to our connection and notify the hub.
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
	authStatus := c.checkAuthStatus(server, message, true)
	if authStatus {
		for i, channel := range c.channels {
			if channel == message.ChannelName {
				c.channels = append(c.channels[:i], c.channels[i+1:]...) // remove the channel.
				break
			}
		}
		server.hub.unbind <- broadcastWriter{conn: c, message: message, peer: false}
	} else {
		log.Println(skittles.BoldRed(fmt.Sprintf("%s: was unable to connect to the channel", c.name))) // remove this at some point.
		c.closeConnection(server)
	}
}

// canWrite checks to make sure we can write.
func (c *connection) canWrite(message *Message, server *Server) bool {
	if c.peer {
		return true
	}
	authStatus := c.checkAuthStatus(server, message, false)
	if authStatus { // check if this a channel the client is bound to
		authStatus = false
		for _, channel := range c.channels {
			if channel == message.ChannelName {
				return true
			}
		}
	}
	return authStatus
}

//closeConnection closes the connection
func (c *connection) closeConnection(server *Server) {
	//unbind from all the channels if client is disconnected
	if server.Notification != nil {
		for _, name := range c.channels {
			server.Notification.UnBindHandler(Message{Name: c.name, Body: "", ChannelName: name, OpCode: UnBindOpCode}, c.token)
		}
	}
	server.hub.unregister <- c
	c.ws.Close()
}

// writePump sends messages from the hub to the websocket connection.
func (c *connection) writePump(server *Server) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.closeConnection(server)
	}()

	for { // blocking loop with select to wait for stimulation.
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, Message{})
				return
			}
			authStatus := c.checkAuthStatus(server, &message, false)
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
