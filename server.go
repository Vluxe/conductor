package conductor

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// ServerClient is the based interface for mocking.
type ServerClient interface {
	Start() error
}

const (
	BindOpCode    = iota // a message to bind to a channel. This will create the channel if it does not exist.
	UnBindOpCode         // a message to unbind from a channel.
	WriteOpCode          // a message to be broadcast on provided channel.
	CleanUpOpcode        // a message to cleanup a disconnected client/connection.
)

//Server is the implementation of ServerClient.
type Server struct {
	Port           int
	h              Hub
	ConnectionAuth ConnectionAuthClient
}

func New(port int, deduper DeDuplication) *Server {
	return &Server{Port: port,
		h:              newMultiPlexHub(deduper),
		ConnectionAuth: &SimpleAuthClient{}}
}

func (s *Server) Start() error {
	go s.h.RunLoop()

	http.HandleFunc("/", s.websocketHandler)

	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil) // need a TLS listener as well.
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if !s.ConnectionAuth.IsValid(r) {
		http.Error(w, "Not authorized", 401)
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

	c := newWSConnection(ws, s.h)
	c.ReadLoop(s.h)
}
