package conductor

import (
	"bytes"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/ugorji/go/codec"
)

const (
	bufferSize = 1024
)

// A client is basic websocket connection.
type Client struct {
	// we hold on to the url for when/if we need to reconnect.
	url *url.URL

	// we hold on to the headers for reconnecting as well.
	headers http.Header

	// the underlining  websocket connection we need to hold on it.
	ws *websocket.Conn

	Read <-chan interface{}
}

// CreateClient allocates and returns a new channel
// ServerUrl is the server url to connect to.
func NewClient(serverUrl string) (*Client, error) {
	u, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}

	header := make(http.Header)
	header.Add("Sec-WebSocket-Protocol", "chat, superchat")
	header.Add("Origin", u.String())

	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		return nil, err
	}
	ws, _, err := websocket.NewClient(conn, u, header, bufferSize, bufferSize)
	if err != nil {
		return nil, err
	}

	channel := make(chan interface{})
	c := &Client{ws: ws, url: u, headers: header, Read: channel}

	go func() {
		for {
			message := decodeMessage(c.ws)
			channel <- message.Body
		}
	}()

	return c, nil
}

func (c *Client) BindToChannel(channelName string) {
	c.write(&Message{OpCode: BindOpCode, ChannelName: channelName})
}

func (c *Client) UnBindFromChannel(channelName string) {
	c.write(&Message{OpCode: UnBindOpCode, ChannelName: channelName})
}

func (c *Client) Write(channelName string, messageBody interface{}) {
	c.write(&Message{OpCode: WriteOpCode, ChannelName: channelName, Body: messageBody})
}

func (c *Client) ServerMessage(messageBody interface{}) {
	c.write(&Message{OpCode: ServerOpCode, ChannelName: "", Body: messageBody})
}

func (c *Client) write(message *Message) {
	buf := new(bytes.Buffer) // probably need a bigger buffer.
	handle := new(codec.MsgpackHandle)
	enc := codec.NewEncoder(buf, handle)
	if err := enc.Encode(message); err != nil {
		log.Fatal(err) // do something else here.
	}

	if err := c.ws.WriteMessage(websocket.BinaryMessage, buf.Bytes()); err != nil {
		log.Fatal(err) // do something else here.
	}

}
