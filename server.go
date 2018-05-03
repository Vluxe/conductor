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

//Server is the implementation of ServerClient.
type Server struct {
	Port           int
	h              Hub
	ConnectionAuth ConnectionAuthClient
}

//New takes in everything need to setup a Server and have all the interfaces implemented.
func New(port int, deduper DeDuplication) *Server {
	return &Server{Port: port,
		h:              newMultiPlexHub(deduper),
		ConnectionAuth: &SimpleAuthClient{}}
}

//Start starts the websocket server to allow connections
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
