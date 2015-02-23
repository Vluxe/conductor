// Copyright 2013/2014 The Gorilla WebSocket & Conductor Authors. All rights reserved.
// some code borrowed from example chat program of
// https://github.com/gorilla/websocket
// Gorilla under BSD-style. See it's LICENSE file for more info.
// Conductor under Apache v2. License can be found in the LICENSE file.

package conductor

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

	// invite to a channel.
	invite chan broadcastWriter
}

//  createHub creates our hub.
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

// run is the runloop that controls all the messages sent within conductor.
func (h *hub) run() {
	for { // blocking loop that waits for stimulation
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
		}
	}
}

// closeConnections removes a connection from the hub.
func (h *hub) closeConnections(c *connection) {
	delete(h.peers, c.name)
	delete(h.clients, c.name)
	for _, name := range c.channels {
		conns := h.channels[name]
		for i, conn := range conns {
			if c == conn {
				h.channels[name] = append(conns[:i], conns[i+1:]...) // remove connection from channel.
				h.processUnbindChannel(name)
			} else {
				conn.send <- Message{Name: c.name, Body: "", ChannelName: name, OpCode: UnBindOpCode}
			}
		}
	}
}

// addConnection adds a connection to the hub.
func (h *hub) addConnection(c *connection) {
	if c.peer {
		if h.peers[c.name] == nil {
			h.peers[c.name] = c
			for name, _ := range h.channels {
				h.channels[name] = append(h.channels[name], c)
			}
		}
	} else if h.clients[c.name] == nil {
		h.clients[c.name] = c
	}
}

// ifConnectionExist checks if the peer has already been added.
func (h *hub) ifConnectionExist(name string) bool {
	if h.peers[name] != nil {
		return true
	}
	return false
}

// bindChannel adds a connection to a channel.
func (h *hub) bindChannel(b broadcastWriter) {
	//we haven't seen this channel before and need to let the peers know
	if !b.conn.peer {
		if h.channels[b.message.ChannelName] == nil {
			for _, c := range h.peers {
				h.channels[b.message.ChannelName] = append(h.channels[b.message.ChannelName], c)
				c.send <- Message{ChannelName: b.message.ChannelName, OpCode: BindOpCode}
			}
		}
	}
	h.channels[b.message.ChannelName] = append(h.channels[b.message.ChannelName], b.conn)
}

// unbindChannel removes a connection from the channel.
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

// processUnbindChannel handles cleaning up a channel.
func (h *hub) processUnbindChannel(channelName string) {
	//delete channel if no longer in use
	if len(h.channels[channelName]) <= len(h.peers) {
		delete(h.channels, channelName)
	}
}

// inviteUser sends an invite to another client. Will also forward to other peers.
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

// broadcastMessage sends a message on a channel.
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
