package conductor

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"net/url"
)

type Client struct {
	conn *websocket.Conn
}

func CreateClient(serverUrl string) (Client, error) {
	u, err := url.Parse(serverUrl)
	bufferSize := 1024
	websocketProtocol := "chat, superchat"
	header := make(http.Header)
	header.Add("Sec-WebSocket-Protocol", websocketProtocol)
	header.Add("Origin", u.String())

	conn, err := net.Dial("tcp", u.Host)
	webConn, _, err := websocket.NewClient(conn, u, header, bufferSize, bufferSize)

	return Client{conn: webConn}, err
}

func (client *Client) Reader() (Message, error) {
	var message Message
	err := client.conn.ReadJSON(&message)
	return message, err
}

func (client *Client) Writer(message *Message) error {
	buffer, err := json.Marshal(message)
	client.conn.WriteMessage(websocket.TextMessage, buffer)
	return err
}
