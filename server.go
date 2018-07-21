package conductor

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// ServerClient is the based interface for mocking.
type ServerClient interface {
	Start(useHTTPServer bool) error
	AddSister(sister SisterClient) error
}

// Server is the implementation of ServerClient.
type Server struct {
	Port     int
	CertName string
	KeyName  string
	Router   http.Handler
	h        Hub
}

// New takes in everything need to setup a Server and have all the interfaces implemented.
// port is what port to bind on.
// deduper is the DeDuplication interface to use message deduplication.
// auther is the ConnectionAuth interface to use for auth.
// storer is the Storage interface to use message storage.
// serverHandler is the ServerHubHandler interface to use for one to one operations.
// sisterManager is the SisterManager interface to use for handling federation.
func New(port int, deduper DeDuplication, auther ConnectionAuth, storer Storage, serverHandler ServerHubHandler, sisterManager SisterManager) *Server {
	return &Server{Port: port,
		h: newMultiPlexHub(deduper, auther, storer, serverHandler, sisterManager)}
}

//Start starts the websocket server to allow connections.
//useHTTPServer is if conductor should start an HTTP server or not.
//Set this to no if you are going to install the WebsocketHandler into your own HTTP system.
func (s *Server) Start(useHTTPServer bool) error {
	go s.h.RunLoop()
	if useHTTPServer {
		http.HandleFunc("/", s.WebsocketHandler)
		// need a better TLS listener. This is really basic.
		if s.CertName != "" && s.KeyName != "" {
			return http.ListenAndServeTLS(fmt.Sprintf(":%d", s.Port), s.CertName, s.KeyName, s.Router)
		}
		return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), s.Router)
	}
	return nil
}

//AddSister adds a sister server to use for federation.
// It sends and receives messages to the other server.
func (s *Server) AddSister(sister SisterClient) error {
	if err := sister.Connect(s.h); err != nil {
		return err
	}
	s.h.SisterManager().addSister(sister)
	go sister.ReadLoop(s.h)
	return nil
}

//WebsocketHandler is the handler of the HTTP HandleFunc. This way you can install conductor into your current HTTP stack.
func (s *Server) WebsocketHandler(w http.ResponseWriter, r *http.Request) {
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
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "failed to upgrade", 500)
		return
	}
	isSister := s.h.Auth().IsSister(r)
	c := newWSConnection(ws, s.h, isSister)
	if s.h.Auth() != nil {
		s.h.Auth().ConnToRequest(r, c)
	}
	if isSister && s.h.SisterManager() != nil {
		s.h.SisterManager().SisterConnected(c)
	}
	c.ReadLoop(s.h)
	if isSister && s.h.SisterManager() != nil {
		s.h.SisterManager().SisterDisconnected(c)
	}
}
