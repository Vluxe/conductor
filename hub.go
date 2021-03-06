package conductor

import (
	"fmt"
)

// ServerHubHandler is the based interface for handling one to one server message between the client and the server.
type ServerHubHandler interface {
	Process(conn Connection, message *Message)
}

// HubConnection is the an interface to hide the other methods of the Hub.
// This way `Connection`s can only write to the Hub and not call its other methods.
type HubConnection interface {
	Write(conn Connection, message *Message)
	ReceivedSisterMessage(conn Connection, message *Message)
}

// Hub is the based interface for what methods aHub should provide.
type Hub interface {
	RunLoop()                                                // This is the master run loop that processes all the messages that come into the channel.
	Write(conn Connection, message *Message)                 // Not sure if I like the duplicate method trick yet...
	Auth() ConnectionAuth                                    // This returns the current auther (if one is used)
	SisterManager() SisterManager                            // This returns the current sister manager (if one is used)
	ReceivedSisterMessage(conn Connection, message *Message) // Handle a sister message into this hub
}

type hubData struct {
	conn     Connection
	message  *Message
	isSister bool
}

// MultiPlexHub is the standard hub that handles interaction between clients and other hubs.
type MultiPlexHub struct {
	// The connections on each channel.
	channels map[string][]Connection

	// The channel we get messages from the hub on.
	messages chan *hubData

	// The deduper implementation to use (if any).
	deduper DeDuplication

	// The authentication implementation to use (if any).
	auther ConnectionAuth

	// The storage implementation to use (if any).
	storer Storage

	// The server handler implementation to use (if any).
	serverHandler ServerHubHandler

	// The sisters registered with the hub
	sisterManager SisterManager
}

func newMultiPlexHub(deduper DeDuplication, auther ConnectionAuth, storer Storage,
	serverHandler ServerHubHandler, sisterManager SisterManager) *MultiPlexHub {
	return &MultiPlexHub{channels: make(map[string][]Connection),
		messages:      make(chan *hubData),
		deduper:       deduper,
		auther:        auther,
		storer:        storer,
		serverHandler: serverHandler,
		sisterManager: sisterManager}
}

// Auth returns the auther object for use in the server.
func (h *MultiPlexHub) Auth() ConnectionAuth {
	return h.auther
}

// SisterManager returns the auther object for use in the server.
func (h *MultiPlexHub) SisterManager() SisterManager {
	return h.sisterManager
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
	h.messages <- &hubData{conn: conn, message: message, isSister: false}
}

// ReceivedSisterMessage is just like Write, expect it sets the isSister flag.
func (h *MultiPlexHub) ReceivedSisterMessage(conn Connection, message *Message) {
	h.messages <- &hubData{conn: conn, message: message, isSister: true}
}

func (h *MultiPlexHub) preProcessHubData(data *hubData) {
	// TODO: validated message is legit here (it has a proper op code, id, etc)
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
	switch opcode := data.message.Opcode; opcode {
	case BindOpcode:
		h.bindConnectionToChannel(data)
	case UnbindOpcode:
		h.unbindConnectionToChannel(data)
	case WriteOpcode:
		h.writeToChannel(data)
	case CleanUpOpcode:
		h.connectionCleanup(data)
	case ServerOpcode:
		h.serverMessage(data)
	case MetaQueryOpcode:
		h.metaQueryMessage(data)
	case MetaQueryResponseOpcode:
		h.handleMetaQueryResponse(data)
	default:
		break
	}
}

func (h *MultiPlexHub) bindConnectionToChannel(data *hubData) {
	if h.auther != nil && !h.auther.CanBind(data.conn, data.message) {
		return //no bind access!
	}
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
	if !data.isSister {
		if h.auther != nil && !h.auther.CanWrite(data.conn, data.message) {
			fmt.Println("blocked unauthorized message")
			return //no write access!
		}
		if h.storer != nil {
			h.storer.Store(data.conn, data.message)
		}
	}

	//send the message to our local clients on this channel
	connections := h.channels[data.message.ChannelName]
	for _, conn := range connections {
		if data.conn == conn {
			continue
		}
		if err := conn.Write(data.message); err != nil {
			fmt.Println(err) // TODO: do something else here.
		} else if h.storer != nil {
			h.storer.SentTo(data.conn, conn, data.message)
		}
	}
	//send the message to the sister servers
	if h.sisterManager != nil {
		h.sisterManager.Write(data.message)
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

func (h *MultiPlexHub) serverMessage(data *hubData) {
	if h.serverHandler != nil {
		h.serverHandler.Process(data.conn, data.message)
	}
}

//Probably going to remove these with the new SWIM stuff.

func (h *MultiPlexHub) metaQueryMessage(data *hubData) {
	if h.sisterManager != nil {
		b := h.sisterManager.MetaQueryResponse()
		data.conn.Write(&Message{Opcode: MetaQueryResponseOpcode, ChannelName: "", Uuid: newUUID(), Body: b})
	}
}

func (h *MultiPlexHub) handleMetaQueryResponse(data *hubData) {
	if h.sisterManager != nil {
		h.sisterManager.HandleMetaQueryResponse(data.message.Body)
	}
}
