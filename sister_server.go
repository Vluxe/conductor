package conductor

import (
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

// SisterServerClient is the based interface for a sister server.
// Server implements this
type SisterServerClient interface {
}

// SisterClient is the based interface for a sister connection.
type SisterClient interface {
	Connect(h HubConnection) error // do the network connection to the sister server
	Write(message *Message)        // write a message to the sister server
	ReadLoop(h HubConnection)      // start the read loop to process messages from the sister server
	//Offer() //offer some kind of security token
	//Answer(token string) return some kind of security token from the token
}

// SisterServer is the standard server that handles interaction between two server and their message hubs.
// NOTE: I can't stress enough how important a good deduper in the hub is.
// This is the only way to not get stuck in an infinite messaging loop bouncing between sister to sister.
type SisterServer struct {
	ServerURL string
	headers   map[string]string
	c         Connection
}

// NewSisterServer creates a new sister server object.
// serverURL is the other server url to connect with.
// h is the hub to write to.
func NewSisterServer(serverURL string, headers map[string]string) *SisterServer {
	return &SisterServer{ServerURL: serverURL, headers: headers}
}

// Connect creates a WebSocket connection to the other server.
func (s *SisterServer) Connect(h HubConnection) error {
	c, err := createWS(s.ServerURL, s.headers, h)
	if err != nil {
		return err
	}
	s.c = c
	return nil
}

// ReadLoop starts the reading loop from the websocket.
// It then reads any incoming messages from other sister servers and processes them accordingly.
func (s *SisterServer) ReadLoop(h HubConnection) {
	s.c.ReadLoop(h)
}

// Write handles taking a message from the hub and sending it back to the server this object represents.
func (s *SisterServer) Write(message *Message) {
	s.c.Write(message)
}

func createWS(serverURL string, headers map[string]string, h HubConnection) (Connection, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	header := make(http.Header)
	header.Add("Sec-WebSocket-Protocol", "chat, superchat")
	header.Add("Origin", u.String())
	if headers != nil {
		for k, v := range headers {
			header.Add(k, v)
		}
	}

	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		return nil, err
	}
	ws, _, err := websocket.NewClient(conn, u, header, bufferSize, bufferSize)
	if err != nil {
		return nil, err
	}
	return newWSConnection(ws, h, true), nil
}
