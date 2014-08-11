package conductor

import (
	"fmt"
	"github.com/acmacalister/skittles"
	"github.com/daltoniam/goguid"
	"github.com/gorilla/websocket"
	"log"
	"net"
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
	Port         int
}

//Creates a new Server struct to work with
func CreateServer(port int) Server {
	return Server{
		hub:  createHub(),
		Port: port,
	}
}

//Starts the server
func (server *Server) Start() {
	log.Println(skittles.Cyan(fmt.Sprintf("starting server on %d...", server.Port)))
	log.Println("uuid is:", guid.NewUUID().String())
	go server.hub.run()
	http.HandleFunc("/", server.websocketHandler)
	err := http.ListenAndServe(fmt.Sprintf(":%d", server.Port), nil)
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
}

//Handles processing new websocket connections to the server.
//This includes peer servers and clients and preforms any auth if implemented
func (server *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	authStatus := true
	if server.Auth != nil {
		authStatus = server.Auth.InitalAuthHandler(r)
	}
	if authStatus {
		peer := false
		name := r.Header.Get("Peer")
		log.Println("peer header", name)
		if name != "" {
			peer = true
			if server.hub.ifConnectionExist(name) {
				log.Println("Already connected to peer:", name)
				http.Error(w, "", http.StatusOK) //not sure if we want this or not
			} else {
				log.Println("Connecting to a new peer:", name)
			}
		} else {
			name = guid.NewGUID().String() //some random name for the clients
		}
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

		c := &connection{send: make(chan Message, 256), ws: ws, peer: peer, peerName: name}
		server.hub.register <- c
		go c.writePump(server)
		c.readPump(server)
	} else {
		http.Error(w, "Failed to connect, access denied", 401)
	}
}

//public function to connect a server to a peer server
func (server *Server) AddPeer(peer string) {
	server.connectToPeer(peer)
}

//connects to a peer server by connecting like a client, but with a special header and message
func (server *Server) connectToPeer(peer string) {
	client, err := CreateClient(peer, guid.NewUUID().String())
	if err != nil {
		log.Println(skittles.BoldRed(err))
		return
	}
	log.Println("connecting to peer named:", peer)
	//need real tokens here...

	//establish we are a peer
	ip := server.getIP()
	selfUrl := fmt.Sprintf("ws://%s:%d", ip, server.Port)
	message := Message{Token: "haha", Name: "", Body: selfUrl, ChannelName: "", OpCode: Peer}
	err = client.Writer(&message)
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
	//go through and bind to all the channels we have on this server to our peer so we get the messages
	for name, _ := range server.hub.channels {
		log.Printf("peer: %s create channel: %s", peer, name)
		message := Message{Token: "haha", Name: "", Body: "", ChannelName: name, OpCode: Bind}
		err = client.Writer(&message)
		if err != nil {
			log.Fatal(skittles.BoldRed(err))
		}
	}
	go client.reader(server)
}

//get our local IP
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

//Forwards messages from peers to own clients
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
