package conductor

type MessageOpCode int

const (
	Create MessageOpCode = 1 << iota
	Delete
	Write
)

// Struct of message json.
type Message struct {
	Token       string        `json:"token"`
	Body        string        `json:"body"`
	ChannelName string        `json:"channel_name"`
	OpCode      MessageOpCode `json:"opcode"`
}
