package conductor

import (
	"crypto/sha1"
	"fmt"
)

type MessageOpCode int

const (
	BindOpCode     MessageOpCode = iota // a message to bind to a channel. This will create the channel if it does not exist.
	UnBindOpCode                        // a message to unbind from a channel.
	WriteOpCode                         // messages sent between clients that should displayed.
	InfoOpCode                          // messages sent between clients that aren't intend to be displayed.
	PeerBindOpCode                      // When peers are connecting.
	PeerOpCode                          // only used between server peers.
	ServerOpCode                        // messages intend to be between a single client and the server (not broadcast).
	InviteOpCode                        // invite between clients to listen on a channel.
)

// Struct of message json.
type Message struct {
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	ChannelName string        `json:"channel_name"`
	OpCode      MessageOpCode `json:"opcode"`
	Additional  interface{}   `json:"additional"`
}

// Creates a sha1 of an authToken.
func HashToken(token string) string {
	data := []byte(token)
	return fmt.Sprintf("%x", sha1.Sum(data))
}
