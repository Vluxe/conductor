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
	// The connections based on channel
	channels map[string][]*connection

	//All registered connections.
	peers map[string]*connection

	// Inbound messages from the connections.
	broadcast chan broadcastWriter

	// Join a new channel
	bind chan broadcastWriter

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
		bind:       make(chan broadcastWriter),
		channels:   make(map[string][]*connection),
		peers:      make(map[string]*connection),
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
		case b := <-h.bind:
			h.bindChannel(b)
		}
	}
}

// closes all the channels in the connection.
func (h *hub) closeConnections(c *connection) {
	delete(h.peers, c.peerName)
	log.Println("closing connection: ", c.peerName)
	for _, name := range c.channels {
		log.Println("Channel: ", name)
		conns := h.channels[name]
		for i, conn := range conns {
			if c == conn {
				h.channels[name] = append(conns[:i], conns[i+1:]...)
				break
			}
		}
	}
}

// add a connection a channel.
func (h *hub) addConnection(c *connection) {
	if c.peer {
		if h.peers[c.peerName] == nil {
			log.Println("adding a new peer", c.peerName)
			h.peers[c.peerName] = c
			for name, _ := range h.channels {
				h.channels[name] = append(h.channels[name], c)
			}
		} else {
			log.Println("peer already bound", c.peerName)
		}
	}
}

// Checks if the peer has already been added
func (h *hub) ifConnectionExist(name string) bool {
	if h.peers[name] != nil {
		return true
	}
	return false
}

//bind to a connection to a channel
func (h *hub) bindChannel(b broadcastWriter) {
	log.Printf("Binding: %s to channel: %s", b.conn.peerName, b.message.ChannelName)
	//we haven't seen this channel before and need to let the peers know
	if !b.conn.peer {
		if h.channels[b.message.ChannelName] == nil {
			for _, c := range h.peers {
				log.Printf("Binding peer: %s to channel: %s", c.peerName, b.message.ChannelName)
				h.channels[b.message.ChannelName] = append(h.channels[b.message.ChannelName], c)
			}
		}
	}
	h.channels[b.message.ChannelName] = append(h.channels[b.message.ChannelName], b.conn)
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
