package conductor

// Storage is the based interface for handling data storage.
type Storage interface {
	Store(conn Connection, message *Message)          //user on this connection wrote a message to a channel.
	SentTo(sender, conn Connection, message *Message) //a connection sent a message to the other connection.
}

// SimpleStorage is the default implmentation of Storage.
// It simply stores the last X messages for each channel.
// You probably shouldn't use this in production.
type SimpleStorage struct {
	channels map[string][]Message
	limit    int
}

// NewSimpleStorage creates a SimpleStorage to use.
// Don't use this for anything but testing
// limit is the amount of messages store for each channel in memory. 100 is a easy default.
func NewSimpleStorage(limit int) *SimpleStorage {
	return &SimpleStorage{channels: make(map[string][]Message),
		limit: limit}
}

// Store puts the X amount of messages in the list
func (s *SimpleStorage) Store(conn Connection, message *Message) {
	//store the messages!
	messages := s.channels[message.ChannelName]
	messages = append(messages, *message)

	if len(messages) > s.limit {
		messages = append(messages[:0], messages[1:]...)
	}
	s.channels[message.ChannelName] = messages
}

// Get retrieves the messages for that channel
func (s *SimpleStorage) Get(channelName string) []Message {
	return s.channels[channelName]
}

// SentTo does nothing in simple storage.
func (s *SimpleStorage) SentTo(sender, conn Connection, message *Message) {
	//noop
}
