// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// some code borrowed from example chat program of
// https://github.com/gorilla/websocket
package conductor

import (
	"log"
)

// hub maintains the set of active connections and broadcasts messages to the
// connections.
type hub struct {
	// Registered connections.
	channels map[string][]*connection

	// Inbound messages from the connections.
	broadcast chan broadcastWriter

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection
}

// create our hub.
func createHub() hub {
	return hub{
		broadcast:  make(chan broadcastWriter),
		register:   make(chan *connection),
		unregister: make(chan *connection),
		channels:   make(map[string][]*connection),
	}
}

// runloop that controls broadcasting messages to other peers.
func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.addConnection(c)
		case c := <-h.unregister:
			h.closeConnections(c)
		case b := <-h.broadcast:
			h.broadcastMessage(b)
		}
	}
}

// closes all the channels in the connection.
func (h *hub) closeConnections(c *connection) {
	log.Println("closing connections...")
	for _, name := range c.channels {
		conns := h.channels[name]
		for i, conn := range conns {
			if c == conn {
				conns = append(conns[:i], conns[i+1:]...)
			}
		}
	}
}

// add a connection a channel.
func (h *hub) addConnection(c *connection) {
	for _, channel := range c.channels {
		log.Println(channel)
		h.channels[channel] = append(h.channels[channel], c)
	}
}

// broadcast message out on channel.
func (h *hub) broadcastMessage(b broadcastWriter) {
	for _, c := range h.channels[b.message.ChannelName] {
		if b.peer && c.peer {
			continue
		}
		if c != b.conn {
			select {
			case c.send <- b.message:
			default:
				h.closeConnections(c)
			}
		}
	}
}
