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

// Server is the implementation of ServerClient.
type Server struct {
	Port int
	h    Hub
}

// New takes in everything need to setup a Server and have all the interfaces implemented.
// The first parameter is what port to bind on.
// The second parameter is the DeDuplication interface to use message deduplication.
// The third parameter is the ConnectionAuth interface to use for auth.
// The fourth parameter is the Storage interface to use message storage.
// The fifth parameter is the ServerHubHandler interface to use for one to one operations.
func New(port int, deduper DeDuplication, auther ConnectionAuth, storer Storage, serverHandler ServerHubHandler) *Server {
	return &Server{Port: port,
		h: newMultiPlexHub(deduper, auther, storer, serverHandler)}
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
	if s.h.Auth() != nil && !s.h.Auth().IsValid(r) {
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
	if s.h.Auth() != nil {
		s.h.Auth().ConnToRequest(r, c)
	}
	c.ReadLoop(s.h)
}
