package conductor

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"net/url"
)

type Client struct {
	conn *websocket.Conn
}

func CreateClient(serverUrl, authToken, peerName string) (Client, error) {
	u, err := url.Parse(serverUrl)
	if err != nil {
		return Client{}, err
	}
	bufferSize := 1024
	websocketProtocol := "chat, superchat"
	header := make(http.Header)
	header.Add("Sec-WebSocket-Protocol", websocketProtocol)
	header.Add("Origin", u.String())
	if peerName != "" {
		header.Add("Peer", peerName)
		header.Add("Token", hashToken(authToken))
	} else {
		header.Add("Token", authToken)
	}

	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		return Client{}, err
	}
	webConn, _, err := websocket.NewClient(conn, u, header, bufferSize, bufferSize)
	if err != nil {
		fmt.Println("boo")
		return Client{}, err
	}
	return Client{conn: webConn}, nil
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
