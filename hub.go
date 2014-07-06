// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// code borrowed from example chat program of
// https://github.com/gorilla/websocket

package conductor

// hub maintains the set of active connections and broadcasts messages to the
// connections.
type hub struct {
	// Registered connections.
	connections map[*connection]bool

	// Inbound messages from the connections.
	broadcast chan broadcastWriter

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection
}

var h = hub{
	broadcast:   make(chan broadcastWriter),
	register:    make(chan *connection),
	unregister:  make(chan *connection),
	connections: make(map[*connection]bool),
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
		case c := <-h.unregister:
			delete(h.connections, c)
			close(c.send)
		case b := <-h.broadcast:
			for c := range h.connections {
				if c != b.conn {
					select {
					case c.send <- b.message:
					default:
						close(c.send)
						delete(h.connections, c)
					}
				}
			}
		}
	}
}
