package conductor

import "encoding/json"

// SisterManagerClient is the based interface for handling adding sisters.
type SisterManagerClient interface {
	addSister(client SisterClient)
}

// SisterManager is the based interface for handling scaling of sister nodes.
// See SimpleMaxSisterManager for more details and a default implementation.
type SisterManager interface {
	Start()                          // start by fetching/searching for your sister servers.
	Write(message *Message)          // write a message to all the sister servers under this management
	SisterConnected(c Connection)    // notification when a sister connects to this server
	SisterDisconnected(c Connection) // notification when a sister disconnects from this server
	MetaQueryResponse() []byte       // respond to a meta query with some content
	HandleMetaQueryResponse([]byte)  // this manager got a response to a query it sent
	addSister(client SisterClient)   // add a sister to the manager
}

// SimpleMaxSisterManager is the implementation of SisterManager.
// It provides scaling of the sisters with a limit of connections you want between each sister.
// For example let's say there are 5 Conductor servers. We can set a max of 2 sisters per server.
// This means each server will connect to 2 other sister servers and balance the connections to ensure redundancy and scale.
// The trick is to balance how much "noise" of duplicate messages being sent versus
// how quickly you want a message to move through the web of connections.
// You also need to consider how many connections from the sisters are open per server.
// More sisters per server means less available sockets for the clients.
type SimpleMaxSisterManager struct {
	possibleSisters  []SisterClient // The sisters that you can connect too
	connectedSisters []SisterClient // The sisters that you are connected too
}

type metaResponse struct {
	Count int `json:"count"`
}

// NewSisterManager is used to create a new SimpleMaxSisterManager
func NewSisterManager() *SimpleMaxSisterManager {
	return &SimpleMaxSisterManager{possibleSisters: []SisterClient{}, connectedSisters: []SisterClient{}}
}

// Start builds a list of servers using a discovery protocol or service (like consult).
// You could also just use a hard set list or config file if you wanted (like this implementation does).
func (s *SimpleMaxSisterManager) Start() {
	s.sendMetaQuery()
	//TODO: kick off the ticker to require and possible rebalance...
}

// SisterConnected adds a server because we got a new sister connection (we might need to requery and balance).
func (s *SimpleMaxSisterManager) SisterConnected(c Connection) {
	//s.sendMetaQuery()
}

// SisterDisconnected removes a server because we lost a sister (we might need to requery and balance).
func (s *SimpleMaxSisterManager) SisterDisconnected(c Connection) {
	//s.sendMetaQuery()
}

// Write forwards this message onto the sisters under its care.
func (s *SimpleMaxSisterManager) Write(message *Message) {
	for _, sister := range s.connectedSisters {
		sister.Write(message)
	}
}

// MetaQueryResponse returns the meta data to use in the response
func (s *SimpleMaxSisterManager) MetaQueryResponse() []byte {
	meta := metaResponse{Count: len(s.connectedSisters)}
	b, _ := json.Marshal(meta)
	return b
}

// HandleMetaQueryResponse handles the response of a meta query
func (s *SimpleMaxSisterManager) HandleMetaQueryResponse(b []byte) {
	var meta metaResponse
	json.Unmarshal(b, &meta)
	//do stuff with meta...
}

// sendMetaQuery asks the sisters for their meta information.
func (s *SimpleMaxSisterManager) sendMetaQuery() {
	for _, sister := range s.possibleSisters {
		sister.Write(&Message{Opcode: MetaQueryOpcode, ChannelName: "", Uuid: newUUID()})
	}
}

// RegisterSister appends the sister server into this hub for broadcasting
func (s *SimpleMaxSisterManager) addSister(client SisterClient) {
	s.possibleSisters = append(s.possibleSisters, client)
}
