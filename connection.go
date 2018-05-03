package conductor

import (
	"time"

	"github.com/gorilla/websocket"
)

// Connection is the based interface for mocking a connection.
type Connection interface {
	Write(message *Message) error // Write is to send a message to the client this connection represents.
	ReadLoop(hub HubConnection)   // ReadLoop is the loop that keeps this connection alive. This is where read messages go.
	Disconnect()                  // Disconnect is use to disconnect the connection.
	Channels() []string
	SetChannels(channels []string)
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 500 // Don't leave me like this!!
)

//WSConnection is the default websocket implementation.
type wsconnection struct {
	// the underlining  websocket connection we need to hold on it.
	ws *websocket.Conn

	// hold onto the ticker so we can clean it up later
	ticker *time.Ticker

	// we might want to change this to a singleton, but maintain a pointer to the hub.
	h HubConnection

	// maintain a list of channels this client is bound to. (useful for clean up.)
	channels []string
}

// newWSConnection creates a new wsconnection object using the gorilla websocket.Conn as the underlying transport.
// HubConnection is also provided to have a simple way to write to the hub without having the hubs runloop methods.
func newWSConnection(ws *websocket.Conn, h HubConnection) *wsconnection {
	return &wsconnection{ws: ws, h: h, channels: make([]string, 1), ticker: time.NewTicker(pingPeriod)}
}

// ReadLoop sets up the websocket reader in a loop to handle messages and forward them to the hub as they come in
// It also starts the ticker to ensure the socket has stimulation and doesn't get closed as an idle connection.
func (c *wsconnection) ReadLoop(hub HubConnection) {
	// Setup our connection's websocket ping/pong handlers from our const values.
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	go c.doTick() // keeps the websocket simulated as per spec.

	for {
		mess := c.decodeMessage()
		if mess == nil {
			c.Disconnect()
			break
		} else {
			hub.Write(c, mess)
		}
	}
}

//Write sends the content of the message to the client.
func (c *wsconnection) Write(message *Message) error {
	buf, err := message.Marshal()
	if err != nil {
		return err
	}
	return c.ws.WriteMessage(websocket.BinaryMessage, buf)
}

func (c *wsconnection) doTick() {
	defer func() {
		c.Disconnect()
	}()

	for { // blocking loop with select to wait for stimulation.
		select {
		case <-c.ticker.C:
			c.ws.WriteMessage(websocket.PingMessage, nil)
		}
	}
}

func (c *wsconnection) Disconnect() {
	c.ticker.Stop()
	c.h.Write(c, &Message{Opcode: CleanUpOpcode, ChannelName: ""})
	c.ws.WriteMessage(websocket.CloseMessage, nil)
	c.ws.Close()
}

func (c *wsconnection) Channels() []string {
	return c.channels
}

func (c *wsconnection) SetChannels(channels []string) {
	c.channels = channels
}

func (c *wsconnection) decodeMessage() *Message {
	_, buf, err := c.ws.ReadMessage()
	if err != nil {
		// handle error
	}
	message, err := Unmarshal(buf)
	if err != nil {
		// handle error
	}
	return message
}
