package conductor

// sisconnection is the sister connection repesentation.
type sisconnection struct {
	// maintain a list of channels this client is bound to. (useful for clean up.)
	channels []string

	// maintain a map of content for the connection (like an auth token so the connection can be associated to a user).
	storage map[string]string
}

// sisconnection stuff
func newSisConnection() *sisconnection {
	return &sisconnection{channels: make([]string, 1)}
}

// ReadLoop does nothing because this is just to model a connection for its channels
func (c *sisconnection) ReadLoop(hub HubConnection) {
	//noop
}

//Store puts something into the local storage of this connection.
func (c *sisconnection) Store(key, value string) {
	c.storage[key] = value
}

//Gets get the value out local storage of this connection.
func (c *sisconnection) Get(key string) string {
	return c.storage[key]
}

//Write sends the content of the message to the client.
func (c *sisconnection) Write(message *Message) error {
	//not sure yet
	return nil
}

// Disconnect does nothing because this is just to model a connection for its channels
func (c *sisconnection) Disconnect() {
}

func (c *sisconnection) Channels() []string {
	return c.channels
}

func (c *sisconnection) SetChannels(channels []string) {
	c.channels = channels
}
