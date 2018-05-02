package conductor

import (
	"bytes"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ugorji/go/codec"
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

	Read <-chan interface{}
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

	channel := make(chan interface{})
	c := &Client{ws: ws, url: u, headers: header, Read: channel}

	go func() {
		for {
			message := c.decodeMessage()
			if message == nil {
				c.ws.Close()
				break
			} else {
				channel <- message.Body
			}
		}
	}()

	return c, nil
}

func (c *Client) BindToChannel(channelName string) {
	id := RandStringBytesMaskImprSrc(32)
	c.write(&Message{OpCode: BindOpCode, ChannelName: channelName, ID: id})
}

func (c *Client) UnBindFromChannel(channelName string) {
	id := RandStringBytesMaskImprSrc(32)
	c.write(&Message{OpCode: UnBindOpCode, ChannelName: channelName, ID: id})
}

func (c *Client) Write(channelName string, messageBody interface{}) {
	id := RandStringBytesMaskImprSrc(32)
	c.write(&Message{OpCode: WriteOpCode, ChannelName: channelName, Body: messageBody, ID: id})
}

func (c *Client) ServerMessage(messageBody interface{}) {
	//c.write(&Message{OpCode: ServerOpCode, ChannelName: "", Body: messageBody})
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

func (c *Client) decodeMessage() *Message {
	_, r, err := c.ws.NextReader()
	if err != nil {
		log.Print("reader error: ", err) // do something else here.
		return nil
	}
	h := new(codec.MsgpackHandle)
	dec := codec.NewDecoder(r, h)
	var message Message
	if err := dec.Decode(&message); err != nil {
		log.Print("decode error: ", err) // do something else here.
		return nil
	}
	return &message
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
