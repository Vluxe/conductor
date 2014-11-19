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

// A Client is basic websocket client.
// It should be created with the CreateClient function to init
// underlining websocket connection.
type Client struct {
	conn *websocket.Conn
}

// CreateClient creates a Client struct and inits the underlining websocket connection.
// param: ServerUrl - URL of conductor (websocket) server.
// param: authToken - Authentication token of conductor (websocket) server.
// param: peerName 	- Name of conductor peer you are connecting to. Empty string if not a peer.
// return: init'ed Client struct if successful, error if failure.
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

// Reads a message for Client. Usually used in a loop.
// returns: Message struct or error.
func (client *Client) Reader() (Message, error) {
	var message Message
	err := client.conn.ReadJSON(&message)
	return message, err
}

// Writes a message to server from Client connection.
// param: message - A conductor Message struct.
// return: error if failure.
func (client *Client) Writer(message *Message) error {
	buffer, err := json.Marshal(message)
	client.conn.WriteMessage(websocket.TextMessage, buffer)
	return err
}
