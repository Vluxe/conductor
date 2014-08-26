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

	//All registered peers.
	peers map[string]*connection

	//All registered connections that aren't peers.
	clients map[string]*connection

	// Inbound messages from the connections.
	broadcast chan broadcastWriter

	// Join a new channel
	bind chan broadcastWriter

	// leave a new channel
	unbind chan broadcastWriter

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection

	// invite to a channel
	invite chan broadcastWriter
}

// create our hub.
func createHub() hub {
	return hub{
		broadcast:  make(chan broadcastWriter),
		register:   make(chan *connection),
		unregister: make(chan *connection),
		bind:       make(chan broadcastWriter),
		unbind:     make(chan broadcastWriter),
		invite:     make(chan broadcastWriter),
		channels:   make(map[string][]*connection),
		peers:      make(map[string]*connection),
		clients:    make(map[string]*connection),
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
		case b := <-h.unbind:
			h.unbindChannel(b)
		case b := <-h.invite:
			h.inviteUser(b)
		case b := <-h.peerbroadcast:
			h.broadcastPeerMessage(b)
		}
	}
}

// closes all the channels in the connection.
func (h *hub) closeConnections(c *connection) {
	delete(h.peers, c.name)
	delete(h.clients, c.name)
	for _, name := range c.channels {
		conns := h.channels[name]
		for i, conn := range conns {
			if c == conn {
				h.channels[name] = append(conns[:i], conns[i+1:]...)
				h.processUnbindChannel(name)
			} else {
				conn.send <- Message{Name: c.name, Body: "", ChannelName: name, OpCode: UnBindOpCode}
			}
		}
	}
}

// add a connection a channel.
func (h *hub) addConnection(c *connection) {
	if c.peer {
		if h.peers[c.name] == nil {
			log.Println("adding a new peer", c.name)
			h.peers[c.name] = c
			for name, _ := range h.channels {
				h.channels[name] = append(h.channels[name], c)
			}
		} else {
			log.Println("peer already bound", c.name)
		}
	} else if h.clients[c.name] == nil {
		h.clients[c.name] = c
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
	//log.Printf("Binding: %s to channel: %s", b.conn.name, b.message.ChannelName)
	//we haven't seen this channel before and need to let the peers know
	if !b.conn.peer {
		if h.channels[b.message.ChannelName] == nil {
			for _, c := range h.peers {
				//log.Printf("Binding peer: %s to channel: %s", c.name, b.message.ChannelName)
				h.channels[b.message.ChannelName] = append(h.channels[b.message.ChannelName], c)
				c.send <- Message{ChannelName: b.message.ChannelName, OpCode: BindOpCode}
			}
		}
	}
	h.channels[b.message.ChannelName] = append(h.channels[b.message.ChannelName], b.conn)
}

//unbind a connection from a channel
func (h *hub) unbindChannel(b broadcastWriter) {
	conns := h.channels[b.message.ChannelName]
	for i, conn := range conns {
		if b.conn == conn {
			h.channels[b.message.ChannelName] = append(conns[:i], conns[i+1:]...)
			h.processUnbindChannel(b.message.ChannelName)
			break
		}
	}
}

//process an unbind on a channel
func (h *hub) processUnbindChannel(channelName string) {
	//delete channel if no longer in use
	if len(h.channels[channelName]) <= len(h.peers) {
		//log.Println("deleting channel:", channelName)
		delete(h.channels, channelName)
	}
}

//invite a user to a channel
func (h *hub) inviteUser(b broadcastWriter) {
	conn := h.clients[b.message.Body]
	if conn != nil {
		conn.send <- *b.message
	}
	//send the invite to the peers in case the user is on another server
	for _, c := range h.peers {
		c.send <- *b.message
	}
}

//broadcast a message to the peers
func (h *hub) broadcastPeerMessage(b broadcastWriter) {
	for _, c := range h.peers {
		if c != b.conn {
			select {
			case c.send <- *b.message:
			}
		}
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
			case c.send <- *b.message:
			default:
				h.closeConnections(c)
			}
		}
	}
}
