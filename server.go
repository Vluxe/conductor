// Copyright 2014 The Conductor Authors. All rights reserved.
// Conductor under Apache v2. License can be found in the LICENSE file.

// Package conductor implements the core server/client pieces of the conductor protocol/system.
//
// This is a work in progress. Conductor is a scalable messaging server written using popular open source tech,
// namely WebSockets and JSON encoding. It aims to be an easy and scalable way to handling real-time connections
// for tens of thousands of clients. See the README for more information.
package conductor

import (
	"fmt"
	"github.com/daltoniam/goguid"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
)

// Implement the Auth interface to satisfy authentication. If unimplemented
// Conductor will ignore authentication altogether.
type Auth interface {
	// InitalAuthHandler runs when a client first connects.
	InitalAuthHandler(r *http.Request, token string) (bool, string)

	// ChannelAuthHandler runs when a client tries to bind to a channel.
	ChannelAuthHandler(message Message, token string) bool

	// MessageAuthHandler runs when a client sends a message.
	MessageAuthHandler(message Message, token string) bool

	// MessageAuthHandler runs when checking if a client must be bound to a channel before writing to it.
	MessagAuthBoundHandler(message Message, token string) bool
}

// Implement the Notification interface to satisfy handling notifications. This interface
// is useful for storing and processing messages between clients (for history, apns, etc).
type Notification interface {
	// PersistentHandler runs on each successful message sent.
	PersistentHandler(message Message, token string)

	// BindHandler runs on each bind message sent.
	BindHandler(message Message, token string)

	// UnBindHandler runs on each unbind message sent.
	UnBindHandler(message Message, token string)

	// InviteHandler runs on each successful  invite message sent.
	InviteHandler(message Message, token string)
}

// // Implement the Notification interface to satisfy handling ServerQuery messages.
// This is used for when a client needs to request something from the server e.g. (channel history, db data, client info, etc).
type ServerQuery interface {
	QueryHandler(message Message, token string) Message
}

// A Server type contains all the handlers, port and authentication token
// that make up a Conductor server.
type Server struct {
	hub          hub
	Auth         Auth
	Notification Notification
	ServerQuery  ServerQuery
	EnablePeers  bool
	Port         int
	AuthToken    string
	guid         string
}

// A Peer type represents a connection to fellow conductor servers.
type Peer struct {
	c     *connection //the connection to the peer.
	sName string      //this server name to send to the peers
}

// CreateServer allocates and returns a new Server.
func CreateServer(port int) Server {
	return Server{
		hub:  createHub(),
		Port: port,
		guid: guid.NewUUID().String(),
	}
}

// Start starts the server to listen for incoming connections.
func (server *Server) Start() error {
	go server.hub.run()

	http.HandleFunc("/", server.websocketHandler)

	return http.ListenAndServe(fmt.Sprintf(":%d", server.Port), nil)
}

// websocketHandler handles processing new WebSocket connections to the server.
func (server *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	peer := false
	peerKey := "Peer"
	tokenKey := "Token"
	name := r.Header.Get(peerKey)
	token := r.Header.Get(tokenKey)
	params := r.URL.Query()
	if len(params[peerKey]) > 0 {
		name = params[peerKey][0]
	}
	if len(params[tokenKey]) > 0 {
		token = params[tokenKey][0]
	}
	if name != "" {
		peer = true
		if server.hub.ifConnectionExist(name) || name == server.guid {
			http.Error(w, "", http.StatusOK) //is this the code we want to send back?
		}
	}
	authStatus := true
	//check if the peers tokens match to make sure they aren't malicious clients
	if peer {
		testToken := hashToken(server.AuthToken)
		if testToken != token {
			authStatus = false
		}

	} else if server.Auth != nil {
		authStatus, name = server.Auth.InitalAuthHandler(r, token)
	}

	if authStatus {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		c := &connection{send: make(chan Message, 256), ws: ws, token: token, peer: peer, name: name}
		server.hub.register <- c
		go c.writePump(server)
		c.readPump(server)
	} else {
		http.Error(w, "Failed to connect, access denied", 401)
	}
}

// AddPeer adds a peer. peer param should be a valid URL/path that this
// server can connect to.
func (server *Server) AddPeer(peer string) {
	server.connectToPeer(peer)
}

//Add a pool of peers. This is a slice of all the peers that can be connected
func (server *Server) AddPeerPool(peers []string) {
	for p := range peers {
		server.connectToPeer(p)
	}
}

//returns all the peers
func (server *Server) AllPeers() []Peer {
	count := len(server.hub.peers)
	var collect [count]Peer
	i := 0
	for _, c := range server.hub.peers {
		collect[i] = Peer{c: c, sName: server.guid}
		i++
	}
	return collect
}

// connectToPeer connects to a peer server by connecting like a client, but with a special header and message.
func (server *Server) connectToPeer(peer string) {
	client, err := CreateClient(peer, server.AuthToken, server.guid)
	if err != nil {
		return
	}

	//establish we are a peer
	ip := server.getIP()
	selfUrl := fmt.Sprintf("ws://%s:%d", ip, server.Port)
	//fmt.Println("got a new peer:", selfUrl)
	client.Write(selfUrl, "", PeerBindOpCode, nil)

	//forwards messages from peers onto to the hub.
	client.ServerBind(func(msg Message) {
		server.hub.broadcast <- broadcastWriter{conn: nil, message: &msg, peer: true}
	})
	go client.ReadLoop()
}

// getIP fetches our local IP.
func (server *Server) getIP() string {
	ipAddrs, _ := net.InterfaceAddrs()
	for _, ipAddr := range ipAddrs {
		ip, _, _ := net.ParseCIDR(ipAddr.String())
		if !ip.IsLoopback() && ip.To4() != nil {
			return ip.String()
		}
	}
	return ""
}

//send the peer a message
func (p *Peer) SendMessage(body, context string) {
	if p.c != nil {
		p.c.send <- Message{Name: p.sName, Body: body, ChannelName: context, OpCode: PeerOpCode}
	}
}
