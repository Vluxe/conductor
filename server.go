package conductor

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

const (
	BindOpCode    = iota // a message to bind to a channel. This will create the channel if it does not exist.
	UnBindOpCode         // a message to unbind from a channel.
	WriteOpCode          // a message to be broadcast on provided channel.
	ServerOpCode         // a message intend to be between a single client and the server (not broadcasted).
	CleanUpOpcode        // a message to cleanup a disconnected client/connection.
)

type Server struct {
	Port int
	h    *hub
}

func New(port int) *Server {
	return &Server{Port: port,
		h: &hub{channels: make(map[string][]*connection),
			messages: make(chan *hubMessage)}}
}

func (s *Server) Start() error {
	go s.h.run()

	http.HandleFunc("/", s.websocketHandler)

	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil) // need a TLS listener as well.
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		//CheckOrigin:     func(r *http.Request) bool { return true },
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "failed to upgrade", 500)
		return
	}

	c := newConnection(ws, s.h)
	c.reader()
}
