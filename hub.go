package conductor

import (
	"bytes"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/ugorji/go/codec"
)

type hubMessage struct {
	c      *connection
	header *Message
}

type hub struct {
	// The connections on each channel.
	channels map[string][]*connection

	// The channel we get messages from the hub on.
	messages chan *hubMessage
}

func (h *hub) run() {
	for { // blocking loop that waits for stimulation
		select {
		case message := <-h.messages:
			if message != nil {
				h.processMessage(message)
			}
		}
	}
}

func (h *hub) processMessage(message *hubMessage) {
	switch opcode := message.header.OpCode; opcode {
	case BindOpCode:
		h.bindConnectionToChannel(message)
	case UnBindOpCode:
		h.unbindConnectionToChannel(message)
	case WriteOpCode:
		h.writeToChannel(message)
	case ServerOpCode:
		h.handleServerMessage(message)
	case CleanUpOpcode:
		h.connectionCleanup(message)
	default:
		fmt.Println("no idea!")
	}
}

func (h *hub) bindConnectionToChannel(message *hubMessage) {
	connections := h.channels[message.header.ChannelName]
	connections = append(connections, message.c)
	h.channels[message.header.ChannelName] = connections
	message.c.channels = append(message.c.channels, message.header.ChannelName)
}

func (h *hub) unbindConnectionToChannel(message *hubMessage) {
	h.removeConnection(message.header.ChannelName, message.c)
	for i, channel := range message.c.channels {
		if channel == message.header.ChannelName {
			message.c.channels = append(message.c.channels[:i], message.c.channels[i+1:]...)
			break
		}
	}
}

func (h *hub) writeToChannel(message *hubMessage) {
	buf := new(bytes.Buffer) // probably need a bigger buffer.
	handle := new(codec.MsgpackHandle)
	enc := codec.NewEncoder(buf, handle)
	if err := enc.Encode(message.header); err != nil {
		log.Fatal(err) // do something else here.
	}

	connections := h.channels[message.header.ChannelName]
	for _, conn := range connections {
		if message.c == conn {
			continue
		}
		if err := conn.ws.WriteMessage(websocket.BinaryMessage, buf.Bytes()); err != nil {
			log.Fatal(err) // do something else here.
		}
	}

}

func (h *hub) handleServerMessage(message *hubMessage) {

}

func (h *hub) connectionCleanup(message *hubMessage) {
	for _, channel := range message.c.channels {
		h.removeConnection(channel, message.c)
	}
}

func (h *hub) removeConnection(channelName string, c *connection) {
	connections := h.channels[channelName]
	for i, conn := range connections {
		if c == conn {
			connections = append(connections[:i], connections[i+1:]...)
			h.channels[channelName] = connections
			break
		}
	}
}
