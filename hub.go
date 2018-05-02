package conductor

import (
	"fmt"
)

// Message represents the framing of the messages that get sent back and forth.
type Message struct {
	OpCode      int         `json:"opcode"`
	ChannelName string      `json:"channel_name"`
	ID          string      `json:"id"`
	Body        interface{} `json:"body"` // don't parse the body, as it defined by the user. *codec.RawExt
}

// HubConnection is the an interface to hide the other methods of the Hub.
// This way `Connection`s can only write to the Hub and not call its other methods.
type HubConnection interface {
	Write(conn Connection, message *Message)
}

// Hub is the based interface for what methods aHub should provide.
type Hub interface {
	RunLoop()                                // This is the master run loop that processes all the messages that come into the channel.
	Write(conn Connection, message *Message) // Not sure if I like the duplicate method trick yet...
}

type hubData struct {
	conn    Connection
	message *Message
}

// MultiPlexHub is the standard hub that handles interaction between clients and other hubs.
type MultiPlexHub struct {
	// The connections on each channel.
	channels map[string][]Connection

	// The channel we get messages from the hub on.
	messages chan *hubData

	// The deduper to use (if any).
	deduper DeDuplication
}

func newMultiPlexHub(deduper DeDuplication) *MultiPlexHub {
	return &MultiPlexHub{channels: make(map[string][]Connection),
		messages: make(chan *hubData),
		deduper:  deduper}
}

// RunLoop is the loop that runs forever processing messages from connections.
func (h *MultiPlexHub) RunLoop() {
	if h.deduper != nil {
		h.deduper.Start()
	}
	for { // blocking loop that waits for stimulation
		select {
		case data := <-h.messages:
			if data != nil {
				h.preProcessHubData(data)
			}
		}
	}
}

// Write is the implementation of HubConnection. This way clients can write messages to the hub without being able to call RunLoop.
func (h *MultiPlexHub) Write(conn Connection, message *Message) {
	h.messages <- &hubData{conn: conn, message: message}
}

func (h *MultiPlexHub) preProcessHubData(data *hubData) {
	// validated message is legit here (it has a proper op code, id, etc)
	if h.deduper != nil {
		if !h.deduper.IsDuplicate(data.message) {
			h.deduper.Add(data.message)
			h.processMessage(data)
		}
	} else {
		h.processMessage(data)
	}
}

func (h *MultiPlexHub) processMessage(data *hubData) {
	switch opcode := data.message.OpCode; opcode {
	case BindOpCode:
		h.bindConnectionToChannel(data)
	case UnBindOpCode:
		h.unbindConnectionToChannel(data)
	case WriteOpCode:
		h.writeToChannel(data)
	case CleanUpOpcode:
		h.connectionCleanup(data)
	default:
		break
	}
}

func (h *MultiPlexHub) bindConnectionToChannel(data *hubData) {
	connections := h.channels[data.message.ChannelName]
	connections = append(connections, data.conn)
	h.channels[data.message.ChannelName] = connections
	data.conn.SetChannels(append(data.conn.Channels(), data.message.ChannelName))
}

func (h *MultiPlexHub) unbindConnectionToChannel(data *hubData) {
	h.removeConnection(data.message.ChannelName, data.conn)
	for i, channel := range data.conn.Channels() {
		if channel == data.message.ChannelName {
			data.conn.SetChannels(append(data.conn.Channels()[:i], data.conn.Channels()[i+1:]...))
			break
		}
	}
}

func (h *MultiPlexHub) writeToChannel(data *hubData) {
	connections := h.channels[data.message.ChannelName]
	for _, conn := range connections {
		if data.conn == conn {
			continue
		}
		if err := conn.Write(data.message); err != nil {
			fmt.Println(err) // do something else here.
		}
	}

}

func (h *MultiPlexHub) connectionCleanup(data *hubData) {
	for _, channel := range data.conn.Channels() {
		h.removeConnection(channel, data.conn)
	}
}

func (h *MultiPlexHub) removeConnection(channelName string, c Connection) {
	connections := h.channels[channelName]
	for i, conn := range connections {
		if c == conn {
			connections = append(connections[:i], connections[i+1:]...)
			h.channels[channelName] = connections
			break
		}
	}
}
