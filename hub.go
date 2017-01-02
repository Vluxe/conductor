package conductor

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

type hubMessage struct {
	c *connection
	m *Message
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
	switch opcode := message.m.Opcode; opcode {
	case BindOpcode:
		h.bindConnectionToChannel(message)
	case UnbindOpcode:
		h.unbindConnectionToChannel(message)
	case WriteOpcode:
		h.writeToChannel(message)
	case ServerOpcode:
		h.handleServerMessage(message)
	case CleanUpOpcode:
		h.connectionCleanup(message)
	default:
		fmt.Println("no idea!")
	}
}

func (h *hub) bindConnectionToChannel(message *hubMessage) {
	connections := h.channels[message.m.StreamName]
	connections = append(connections, message.c)
	h.channels[message.m.StreamName] = connections
	message.c.channels = append(message.c.channels, message.m.StreamName)
}

func (h *hub) unbindConnectionToChannel(message *hubMessage) {
	h.removeConnection(message.m.StreamName, message.c)
	for i, channel := range message.c.channels {
		if channel == message.m.StreamName {
			message.c.channels = append(message.c.channels[:i], message.c.channels[i+1:]...)
			break
		}
	}
}

func (h *hub) writeToChannel(message *hubMessage) {
	buf, err := message.m.Marshal()
	if err != nil {
		log.Fatal(err)
	}

	connections := h.channels[message.m.StreamName]
	for _, conn := range connections {
		if message.c == conn {
			continue
		}
		if err := conn.ws.WriteMessage(websocket.BinaryMessage, buf); err != nil {
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
