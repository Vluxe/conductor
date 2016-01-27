package conductor

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ugorji/go/codec"
)

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

type connection struct {
	// the underlining  websocket connection we need to hold on it.
	ws *websocket.Conn

	// we might want to change this to a singleton, but maintain a pointer to the hub.
	h *hub

	// maintain a list of channels this client is bound to. (useful for clean up.)
	channels []string
}

type Message struct {
	OpCode      int         `json:"opcode"`
	ChannelName string      `json:"channel_name"`
	Body        interface{} `json:"body"` // don't parse the body, as it defined by the user. *codec.RawExt
}

func (c *connection) reader() {
	// Setup our connection's websocket ping/pong handlers from our const values.
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	go c.ticker() // keeps the websocket simulated as per spec.

	for {
		c.h.messages <- c.decodeHubMessage()
	}
}

func (c *connection) ticker() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		//c.h.messages <- &hubMessage{c: c, header: &Message{OpCode: LeaveOpCode, ChannelName: ""}}
		c.ws.WriteMessage(websocket.CloseMessage, nil)
	}()

	for { // blocking loop with select to wait for stimulation.
		select {
		case <-ticker.C:
			c.ws.WriteMessage(websocket.PingMessage, nil)
		}
	}
}

func (c *connection) decodeHubMessage() *hubMessage {
	header := decodeMessage(c.ws)
	return &hubMessage{c: c, header: &header}
}

func decodeMessage(ws *websocket.Conn) Message {
	_, r, err := ws.NextReader()
	if err != nil {
		log.Fatal("reader error: ", err) // do something else here.
	}
	h := new(codec.MsgpackHandle)
	dec := codec.NewDecoder(r, h)
	var header Message
	if err := dec.Decode(&header); err != nil {
		log.Fatal("decode error: ", err) // do something else here.
	}
	return header
}
