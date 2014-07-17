package conductor

import (
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
}

func CreateServer() Server {
	return Server{
		hub: createHub(),
	}
}

func (server *Server) Start() {
	log.Println(skittles.Cyan("starting server..."))
	go server.hub.run()
	http.HandleFunc("/", server.websocketHandler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
}

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

		c := &connection{send: make(chan Message, 256), ws: ws}
		server.hub.register <- c
		go c.writePump(server)
		c.readPump(server)
	} else {
		http.Error(w, "Failed ", 401)
	}
}
