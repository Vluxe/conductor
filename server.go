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
}

// CreateServer allocates and returns a new Server.
func CreateServer(port int) Server {
	return Server{
		hub:  createHub(),
		Port: port,
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
		if server.hub.ifConnectionExist(name) {
			http.Error(w, "", http.StatusOK) //not sure if we want this or not
		}
	}
	authStatus := true
	//check if the peers tokens match to make sure they are malicious clients
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

// connectToPeer connects to a peer server by connecting like a client, but with a special header and message.
func (server *Server) connectToPeer(peer string) {
	client, err := CreateClient(peer, server.AuthToken, guid.NewUUID().String())
	if err != nil {
		return
	}

	//establish we are a peer
	ip := server.getIP()
	selfUrl := fmt.Sprintf("ws://%s:%d", ip, server.Port)
	message := Message{Body: selfUrl, ChannelName: "", OpCode: PeerBindOpCode}
	client.Writer(&message)
	go client.reader(server)
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

// reader forwards messages from peers onto to the hub.
func (client *Client) reader(server *Server) {
	for {
		message, err := client.Reader()
		if err != nil {
			break
		}
		server.hub.broadcast <- broadcastWriter{conn: nil, message: &message, peer: true}
	}
}
