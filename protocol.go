// Copyright 2014 The Conductor Authors. All rights reserved.
// Conductor under Apache v2. License can be found in the LICENSE file.

package conductor

import (
	"crypto/sha1"
	"fmt"
)

const (
	BindOpCode     = 1 // a message to bind to a channel. This will create the channel if it does not exist.
	UnBindOpCode   = 2 // a message to unbind from a channel.
	WriteOpCode    = 3 // messages sent between clients that should displayed.
	InfoOpCode     = 4 // messages sent between clients that aren't intend to be displayed.
	PeerBindOpCode = 5 // When peers are connecting.
	PeerOpCode     = 6 // only used between server peers.
	ServerOpCode   = 7 // messages intend to be between a single client and the server (not broadcast).
	InviteOpCode   = 8 // invite between clients to listen on a channel.
)

// Message is the struct of a JSON message within conductor.
type Message struct {
	Name        string      `json:"name"`
	Body        string      `json:"body"`
	ChannelName string      `json:"channel_name"`
	OpCode      int         `json:"opcode"`
	Additional  interface{} `json:"additional"`
}

// hashToken creates a sha1 of an authToken.
func hashToken(token string) string {
	data := []byte(token)
	return fmt.Sprintf("%x", sha1.Sum(data))
}
