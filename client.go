// Copyright 2014 The Conductor Authors. All rights reserved.
// Conductor under Apache v2. License can be found in the LICENSE file.

package conductor

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"net/url"
)

// A Client is basic websocket connection.
type Client struct {
	conn *websocket.Conn
}

// CreateClient allocates and returns a new Client connection.
// ServerUrl is the server url to connect to. authToken is
// your authentication token. peerName if you are connecting to a peer. Normally just a blank string.
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

// Reader reads a nessage from the client connection. Usually used in a loop.
func (client *Client) Reader() (Message, error) {
	var message Message
	err := client.conn.ReadJSON(&message)
	return message, err
}

// Writer writes a message on the client connection.
func (client *Client) Writer(message *Message) error {
	buffer, err := json.Marshal(message)
	client.conn.WriteMessage(websocket.TextMessage, buffer)
	return err
}
