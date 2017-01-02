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
			if message == nil {
				break
			} else {
				channel <- message.Body
			}
		}
	}()

	return c, nil
}

func (c *Client) Bind(streamName string) {
	c.write(&Message{Opcode: BindOpcode, StreamName: streamName})
}

func (c *Client) Unbind(streamName string) {
	c.write(&Message{Opcode: UnbindOpcode, StreamName: streamName})
}

func (c *Client) Write(streamName string, messageBody []byte) {
	c.write(&Message{Opcode: WriteOpcode, StreamName: streamName, Body: messageBody})
}

func (c *Client) ServerMessage(messageBody []byte) {
	c.write(&Message{Opcode: ServerOpcode, StreamName: "", Body: messageBody})
}

func (c *Client) WriteStream(streamName string, reader io.Reader) error {
	buf := make([]byte, 32*1024)
	c.write(&Message{Opcode: StreamStartOpcode, StreamName: streamName})
	defer c.write(&Message{Opcode: StreamEndOpcode, StreamName: streamName})
	for {
		nr, err := reader.Read(buf)
		if nr > 0 {
			c.write(&Message{Opcode: StreamWriteOpcode, StreamName: streamName, Body: buf[0:nr]})
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
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
