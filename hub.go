// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// code borrowed from example chat program of
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

var h = hub{
	broadcast:  make(chan broadcastWriter),
	register:   make(chan *connection),
	unregister: make(chan *connection),
	channels:   make(map[string][]*connection),
}

// runloop that controls broadcasting messages to other peers.
func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.addConnection(c)
		case c := <-h.unregister:
			log.Println("closing connections...")
			closeConnections(c, h)
		case b := <-h.broadcast:
			for _, c := range h.channels[b.channelUUID] {
				if c != b.conn {
					select {
					case c.send <- b.message:
					default:
						closeConnections(c, h)
					}
				}
			}
		}
	}
}

// closes all the channels in the connection.
func closeConnections(c *connection, h *hub) {
	//close(c.send) trying to close a closed channel...
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
		h.channels[channel] = append(h.channels[channel], c)
	}
}
