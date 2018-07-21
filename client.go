package conductor

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

const (
	bufferSize = 1024
)

//Client is basic websocket connection.
type Client struct {
	// we hold on to the url for when/if we need to reconnect.
	url *url.URL

	// we hold on to the headers for reconnecting as well.
	headers http.Header

	// the underlining  websocket connection we need to hold on it.
	ws *websocket.Conn

	Read <-chan *Message
}

// NewClient allocates and returns a new channel
// ServerUrl is the server url to connect to.
func NewClient(serverURL string) (*Client, error) {
	u, err := url.Parse(serverURL)
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

	channel := make(chan *Message)
	c := &Client{ws: ws, url: u, headers: header, Read: channel}

	go func() {
		for {
			message := c.decodeMessage()
			if message == nil {
				c.ws.Close()
				break
			} else {
				channel <- message
			}
		}
	}()

	return c, nil
}

//Bind is used to send a bind request to a channel
func (c *Client) Bind(channelName string) {
	c.write(&Message{Opcode: BindOpcode, ChannelName: channelName, Uuid: newUUID()})
}

//Unbind is used to send an unbind request to a channel
func (c *Client) Unbind(channelName string) {
	c.write(&Message{Opcode: UnbindOpcode, ChannelName: channelName, Uuid: newUUID()})
}

//Write to send a message to a channel
func (c *Client) Write(channelName string, messageBody []byte) {
	c.write(&Message{Opcode: WriteOpcode, ChannelName: channelName, Uuid: newUUID(), Body: messageBody})
}

//ServerMessage sends a message to the server for server operations (like getting message history or something)
func (c *Client) ServerMessage(messageBody []byte) {
	c.write(&Message{Opcode: ServerOpcode, ChannelName: "", Uuid: newUUID(), Body: messageBody})
}

//WriteStream is to write an whole file to the stream. It chucks the data using the special stream op codes.
func (c *Client) WriteStream(channelName string, reader io.Reader) error {
	buf := make([]byte, 32*1024)
	c.write(&Message{Opcode: StreamStartOpcode, ChannelName: channelName, Uuid: newUUID()})
	defer c.write(&Message{Opcode: StreamEndOpcode, ChannelName: channelName, Uuid: newUUID()})
	for {
		nr, err := reader.Read(buf)
		if nr > 0 {
			c.write(&Message{Opcode: StreamWriteOpcode, ChannelName: channelName, Uuid: newUUID(), Body: buf[0:nr]})
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func (c *Client) write(message *Message) {
	buf, err := message.Marshal()
	if err != nil {
		log.Fatal(err)
	}
	if err := c.ws.WriteMessage(websocket.BinaryMessage, buf); err != nil {
		log.Fatal(err) // do something else here.
	}

}

func (c *Client) decodeMessage() *Message {
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
