package conductor

import (
	"time"

	"github.com/gorilla/websocket"
)

// Connection is the based interface for mocking a connection.
type Connection interface {
	Write(message *Message) error  // Write is to send a message to the client this connection represents.
	ReadLoop(hub HubConnection)    // ReadLoop is the loop that keeps this connection alive. Don't call this.
	Disconnect()                   // Disconnect is use to disconnect the connection.
	Channels() []string            // Channels is to hold the channels this connection is bound to. Very useful for auth and cleanup.
	SetChannels(channels []string) // Update the channel list of this connection.
	Store(key, value string)       // Store is a map of local storage for the connection. This way you can identify the connection in other interfaces.
	Get(key string) string         // Get is a map of local storage for the connection.
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 500 //TODO: Don't leave me like this!!
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

	// maintain a map of content for the connection (like an auth token so the connection can be associated to a user).
	storage map[string]string

	// is this a connection used for sister federation between servers?
	isSister bool
}

// newWSConnection creates a new wsconnection object using the gorilla websocket.Conn as the underlying transport.
// HubConnection is also provided to have a simple way to write to the hub without having the hubs runloop methods.
func newWSConnection(ws *websocket.Conn, h HubConnection, isSister bool) *wsconnection {
	return &wsconnection{ws: ws, h: h, channels: make([]string, 1), ticker: time.NewTicker(pingPeriod),
		isSister: isSister, storage: make(map[string]string)}
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
			if c.isSister {
				hub.ReceivedSisterMessage(c, mess)
			} else {
				hub.Write(c, mess)
			}
		}
	}
}

//Store puts something into the local storage of this connection.
func (c *wsconnection) Store(key, value string) {
	c.storage[key] = value
}

//Gets get the value out local storage of this connection.
func (c *wsconnection) Get(key string) string {
	return c.storage[key]
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
		//TODO: handle error
	}
	message, err := Unmarshal(buf)
	if err != nil {
		//TODO: handle error
	}
	return message
}
