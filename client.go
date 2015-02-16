// Copyright 2014 The Conductor Authors. All rights reserved.
// Conductor under Apache v2. License can be found in the LICENSE file.

package conductor

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"net/url"
)

const (
	kAllMessages = "*"
	bufferSize   = 1024
)

// A Client is basic websocket connection.
type Client struct {
	url           *url.URL    //store this for reconnect
	headers       http.Header //store this for reconnect
	conn          *websocket.Conn
	isConnected   bool
	channels      map[string]func(Message)
	serverChannel func(Message)
	isPeer        bool
}

// CreateClient allocates and returns a new Client connection.
// ServerUrl is the server url to connect to. authToken is
// your authentication token. peerName if you are connecting to a peer. Normally just a blank string.
func CreateClient(serverUrl, authToken, peerName string) (Client, error) {
	u, err := url.Parse(serverUrl)
	if err != nil {
		return Client{}, err
	}

	websocketProtocol := "chat, superchat"
	header := make(http.Header)
	header.Add("Sec-WebSocket-Protocol", websocketProtocol)
	header.Add("Origin", u.String())
	isPeer := false
	if peerName != "" {
		header.Add("Peer", peerName)
		header.Add("Token", hashToken(authToken))
		isPeer = true
	} else {
		header.Add("Token", authToken)
	}

	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		return Client{}, err
	}
	webConn, _, err := websocket.NewClient(conn, u, header, bufferSize, bufferSize)
	if err != nil {
		return Client{}, err
	}
	c := Client{conn: webConn, url: u, headers: header, channels: make(map[string]func(Message))}
	c.isConnected = true
	c.isPeer = isPeer
	return c, nil
}

//Bind to a channel by its name and get messages from it
func (client *Client) Bind(channelName string, messages func(Message)) {
	client.channels[channelName] = messages
	if channelName != kAllMessages {
		client.Write("", channelName, BindOpCode, nil)
	}
}

//Unbind from a channel by its name and stop getting messages from it
func (client *Client) Unbind(channelName string) {
	delete(client.channels, channelName)
	if channelName != kAllMessages {
		client.Write("", channelName, UnBindOpCode, nil)
	}
}

//Bind to the "server channel" and get messages that are from of the server opcode
func (client *Client) ServerBind(messages func(Message)) {
	client.serverChannel = messages
}

//UnBind to the "server channel" and stop getting messages that are of the server opcode
func (client *Client) ServerUnbind() {
	client.serverChannel = nil
}

//send a message to a channel with the write opcode
func (client *Client) SendMessage(body, channelName string, additional interface{}) {
	client.Write(body, channelName, WriteOpCode, additional)
}

//send a message to a channel with the info opcode
func (client *Client) SendInfo(body, channelName string, additional interface{}) {
	client.Write(body, channelName, InfoOpCode, additional)
}

//send a invite to a channel to a user
func (client *Client) SendInvite(name, channelName string, additional interface{}) {
	client.Write(name, channelName, InviteOpCode, additional)
}

//send a message to a channel with the server opcode.
//note that channelName is optional in this case and is only used for context.
func (client *Client) SendServerMessage(body, channelName string, additional interface{}) {
	client.Write(body, channelName, ServerOpCode, additional)
}

//writes a message to the websocket
func (client *Client) Write(body, channelName string, opcode int, additional interface{}) error {
	//write a message to the socket
	msg := Message{Body: body, ChannelName: channelName, OpCode: opcode, Additional: additional}
	buffer, err := json.Marshal(msg)
	client.conn.WriteMessage(websocket.TextMessage, buffer)
	return err
}

//read loop to listen for messages
func (client *Client) ReadLoop() {
	for client.isConnected {
		var message Message
		err := client.conn.ReadJSON(&message)
		if err != nil {
			//need to do notify that an error has occurred
			break
		}
		if client.isPeer && client.serverChannel != nil {
			client.serverChannel(message)
		} else {
			//process the message
			if message.OpCode == ServerOpCode || message.OpCode == InviteOpCode {
				if client.serverChannel != nil {
					client.serverChannel(message)
				}
			} else {
				if client.channels[message.ChannelName] != nil {
					callback := client.channels[message.ChannelName]
					callback(message)
				}
				if client.channels[kAllMessages] != nil {
					callback := client.channels[kAllMessages]
					callback(message)
				}
			}
		}
	}
}

//connect to the stream, if not connected
func (client *Client) Connect() error {
	if !client.isConnected {
		client.channels = make(map[string]func(Message))
		conn, err := net.Dial("tcp", client.url.Host)
		if err != nil {
			return err
		}
		webConn, _, err := websocket.NewClient(conn, client.url, client.headers, bufferSize, bufferSize)
		if err != nil {
			return err
		}
		client.conn = webConn
	}
	return nil
}

//disconnect from the stream, if connected
func (client *Client) Disconnect() {
	if client.isConnected {
		client.channels = make(map[string]func(Message))
		client.conn.Close()
	}
}
