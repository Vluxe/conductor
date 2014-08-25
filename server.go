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

// This is used for authenication
type auth interface {
	InitalAuthHandler(r *http.Request, token string) (bool, string)
	ChannelAuthHandler(message Message, token string) bool
	MessageAuthHandler(message Message, token string) bool
}

//This is used for storing messages between clients (for history and such)
type notification interface {
	PersistentHandler(message Message, token string)
}

//This is use for message between a client and the server
type serverQuery interface {
	QueryHandler(message Message, token string) Message
}

//The server struct that has all the magic
type Server struct {
	hub          hub
	Auth         auth
	Notification notification
	ServerQuery  serverQuery
	EnablePeers  bool
	Port         int
	AuthToken    string
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
	//log.Println("uuid is:", guid.NewUUID().String())
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
			log.Println("Already connected to peer:", name)
			http.Error(w, "", http.StatusOK) //not sure if we want this or not
		}
	}
	authStatus := true
	//check if the peers tokens match to make sure they are malicious clients
	if peer {
		testToken := HashToken(server.AuthToken)
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
			if _, ok := err.(websocket.HandshakeError); !ok {
				log.Println(err)
			}
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

//public function to connect a server to a peer server
func (server *Server) AddPeer(peer string) {
	server.connectToPeer(peer)
}

//connects to a peer server by connecting like a client, but with a special header and message
func (server *Server) connectToPeer(peer string) {
	client, err := CreateClient(peer, server.AuthToken, guid.NewUUID().String())
	if err != nil {
		log.Println(skittles.BoldRed(err))
		return
	}
	log.Println("connecting to peer named:", peer)

	//establish we are a peer
	ip := server.getIP()
	selfUrl := fmt.Sprintf("ws://%s:%d", ip, server.Port)
	message := Message{Body: selfUrl, ChannelName: "", OpCode: PeerBindOpCode}
	err = client.Writer(&message)
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
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
		server.hub.broadcast <- broadcastWriter{conn: nil, message: &message, peer: true}
	}
}
