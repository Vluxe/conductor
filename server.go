package conductor

import (
	"fmt"
	"github.com/acmacalister/skittles"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type auth interface {
	InitalAuthHandler(r *http.Request) bool
	ChannelAuthHandler() bool
	MessageAuthHandler() bool
}

type notification interface {
	PersistentHandler()
}

type Server struct {
	hub          hub
	Auth         auth
	Notification notification
	EnablePeers  bool
	peers        []string
	Port         int
}

func CreateServer(port int) Server {
	return Server{
		hub:  createHub(),
		Port: port,
	}
}

func (server *Server) Start() {
	log.Println(skittles.Cyan(fmt.Sprintf("starting server on %d...", server.Port)))
	go server.hub.run()
	http.HandleFunc("/", server.websocketHandler)
	err := http.ListenAndServe(fmt.Sprintf(":%d", server.Port), nil)
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
}

func (server *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	peer := false

	if r.Header["Peer"] != nil {
		peer = true
	}

	authStatus := true
	if server.Auth != nil {
		authStatus = server.Auth.InitalAuthHandler(r)
	}

	if authStatus {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				log.Println(err)
			}
			return
		}

		c := &connection{send: make(chan Message, 256), ws: ws, peer: peer}
		server.hub.register <- c
		go c.writePump(server)
		c.readPump(server)
	} else {
		http.Error(w, "Failed ", 401)
	}
}

func (server *Server) AddPeer(peer string) {
	server.connectToPeer(peer)
}

func (server *Server) LoadPeers(peers []string) {
	server.peers = append(server.peers, peers...)
	server.connectToPeers()
}

func (server *Server) connectToPeer(peer string) {
	client, err := CreateClient(peer, true)
	if err != nil {
		log.Println(skittles.BoldRed(err))
		return
	}
	server.peers = append(server.peers, peer)
	message := Message{Token: "haha", Name: "a peer...", Body: fmt.Sprintf("ws://localhost:%d", server.Port), ChannelName: "hello", OpCode: Info}
	err = client.Writer(&message)
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
	message = Message{Token: "haha", Name: "channel", Body: "binding...", ChannelName: "hello", OpCode: Bind}
	err = client.Writer(&message)
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
	go client.reader(server)
}

func (server *Server) connectToPeers() {
	for _, peer := range server.peers {
		server.connectToPeer(peer)
	}
}

func (client *Client) reader(server *Server) {
	for {
		message, err := client.Reader()
		if err != nil {
			log.Println(skittles.BoldRed(err))
			break
		}

		server.hub.broadcast <- broadcastWriter{conn: nil, message: message, peer: true}
	}
}
